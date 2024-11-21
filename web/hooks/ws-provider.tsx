import { Button } from "@/components/ui/button";
import {
  createChatMessagePacket,
  createPacket,
  Packet,
  PacketSchema,
  PacketType,
  ReadyState,
  WS,
} from "@/lib/ws";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { Outlet } from "react-router-dom";
import { useReceiveMessage } from "./chats";
import {
  Message,
  MessageCreateRequest,
  MessageSchema,
  MessageStatus,
  MessageStatusUpdate,
  MessageStatusUpdateSchema,
  ReadRoomMessagePacketPayloadSchema,
  TypingEventPacketPayload,
  TypingEventPacketPayloadSchema,
} from "@/types/chat";
import { useSWRConfig } from "swr";
import { useSession } from "@/components/providers/session-provider";

type WSContext = {
  sendMessage: (payload: MessageCreateRequest) => void;
  readMessage: (roomID: string) => void;
  emitTypingEvent: (roomID: string, typing: boolean) => void;
  typing: Record<string, string[]>;
};

const wsContext = createContext<WSContext>({
  sendMessage: () => {},
  readMessage: () => {},
  emitTypingEvent: () => {},
  typing: {},
});

export const useWS = () => useContext(wsContext);

export default function WSProvider() {
  const ws = useRef<WS>(
    new WS({
      onReadStateChange: (readyState) => setReadyState(readyState),
      onPacketReceived: (packet) => handlePacketReceived(packet),
    })
  );

  const [readyState, setReadyState] = useState(ReadyState.Connecting);
  const [typing, setTyping] = useState<Record<string, string[]>>({});
  const { trigger } = useReceiveMessage();
  const { mutate } = useSWRConfig();
  const session = useSession();

  const emitTypingEvent = (roomID: string, typing: boolean) => {
    const payload: TypingEventPacketPayload = {
      roomID: roomID,
      typing: typing,
      user: session.username,
    };
    const packet = createPacket(
      PacketType.TypingEvent,
      JSON.stringify(payload)
    );
    ws.current.sendPacket(packet);
  };

  const onReadRoomMessages = (roomID: string) => {
    mutate(
      `/api/chats/rooms/${roomID}/messages`,
      async (currentMessages: Message[] | undefined) => {
        if (!currentMessages) return [];
        return currentMessages.map((message) => {
          if (message.status === MessageStatus.SENT) {
            return { ...message, status: MessageStatus.READ };
          }
          return message;
        });
      },
      { revalidate: false }
    );
  };

  const readMessage = useCallback(
    (roomID: string) => {
      const packet = createPacket(
        PacketType.ReadRoomMessages,
        JSON.stringify({ roomID, readBy: session.username })
      );
      ws.current.sendPacket(packet);
      mutate(
        `/api/chats/rooms/${roomID}/messages`,
        async (currentMessages: Message[] | undefined) => {
          if (!currentMessages) return [];
          return currentMessages.map((message) => {
            if (message.status === MessageStatus.SENT) {
              console.log("readMessage", message);
              return { ...message, status: MessageStatus.READ };
            }
            return message;
          });
        }
      );
    },
    [ws.current, mutate]
  );

  const onReceivedUpdateMessageStatus = (
    correlationID: number,
    { messageID, status, roomID }: MessageStatusUpdate
  ) => {
    mutate(
      `/api/chats/rooms/${roomID}/messages`,
      async (currentMessages: Message[] | undefined) => {
        if (!currentMessages) return [];
        return currentMessages.map((message) => {
          if (
            typeof message.correlationID === "number" &&
            message.correlationID === correlationID
          ) {
            console.log("updating message status", messageID, status);
            return { ...message, status, id: messageID };
          }
          return message;
        });
      },
      { revalidate: false }
    );
  };

  const sendMessage = (message: MessageCreateRequest) => {
    const packet = createChatMessagePacket(message);
    ws.current.sendPacket(packet);
    const newMessage: Message = {
      id: "",
      correlationID: packet.correlationID,
      data: message.data,
      type: message.type,
      sender: session.username,
      sentAt: new Date().toISOString(),
      roomID: message.roomID,
      status: MessageStatus.PENDING,
    };
    mutate(
      `/api/chats/rooms/${message.roomID}/messages`,
      async (currentMessages: Message[] | undefined) => [
        newMessage,
        ...(currentMessages ? currentMessages : []),
      ],
      {
        revalidate: false,
      }
    );
  };

  const handleTypingEvent = (payload: TypingEventPacketPayload) => {
    if (payload.typing) {
      setTyping((prev) => {
        if (prev[payload.roomID]) {
          const users = prev[payload.roomID];
          return {
            ...prev,
            [payload.roomID]: [...users, payload.user],
          };
        } else {
          return {
            ...prev,
            [payload.roomID]: [payload.user],
          };
        }
      });
    } else {
      setTyping((prev) => {
        if (prev[payload.roomID]) {
          const users = prev[payload.roomID];
          return {
            ...prev,
            [payload.roomID]: users.filter((u) => u !== payload.user),
          };
        } else {
          return prev;
        }
      });
    }
  };

  const handlePacketReceived = (packet: Packet) => {
    let parsedMessage = {};
    try {
      parsedMessage = JSON.parse(packet.data);
    } catch (e) {
      console.log("error parsing message", e);
    }
    switch (packet.type) {
      case PacketType.ChatMessage:
        const messageValidation = MessageSchema.safeParse(parsedMessage);
        if (messageValidation.success) {
          trigger(messageValidation.data);
        } else {
          console.log(
            "invalid message received",
            "\ndata",
            packet.data,
            "\nerror",
            messageValidation.error.format()
          );
        }
        break;

      case PacketType.ChatMessageStatusUpdate:
        console.log("message status update received", parsedMessage);
        const validation = MessageStatusUpdateSchema.safeParse(parsedMessage);
        if (validation.success) {
          onReceivedUpdateMessageStatus(packet.correlationID, validation.data);
        } else {
          console.log(
            "invalid message status update received",
            "\ndata",
            packet.data,
            "\nerror",
            validation.error.format()
          );
        }
        break;

      case PacketType.ReadRoomMessages:
        const v = ReadRoomMessagePacketPayloadSchema.safeParse(parsedMessage);
        if (v.success) {
          onReadRoomMessages(v.data.roomID);
        }
        console.log(v);
        break;

      case PacketType.TypingEvent:
        const vv = TypingEventPacketPayloadSchema.safeParse(parsedMessage);
        if (vv.success) {
          handleTypingEvent(vv.data);
        }
        break;

      default:
        console.log("Cannot handle packet", packet);
    }
  };

  useEffect(() => {
    ws.current.connect();
    setReadyState(ws.current.readyState);
    return () => {
      console.log("app close connection");
      ws.current.close();
    };
  }, []);

  if (readyState === ReadyState.Connecting)
    return (
      <div className="w-screen h-screen flex items-center justify-center">
        Connecting...
      </div>
    );

  if (readyState === ReadyState.Closed) {
    return (
      <div className="w-screen h-screen flex items-center justify-center">
        <p>Connection closed</p>
        <Button onClick={() => ws.current.connect()}>Reconnect</Button>
      </div>
    );
  }

  return (
    <wsContext.Provider
      value={{ sendMessage, readMessage, emitTypingEvent, typing }}
    >
      <Outlet />
    </wsContext.Provider>
  );
}

export const useTyping = (roomID: string) => {
  const { typing } = useWS();
  return typing[roomID] || [];
};
