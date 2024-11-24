import { Button } from "@/components/ui/button";
import { ReadyState, WS } from "@/lib/ws";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { Outlet } from "react-router-dom";
import { Message, MessageType, Room } from "@/types/chat";
import { useSWRConfig } from "swr";
import { useSession } from "@/components/providers/session-provider";
import {
  BroadcastReadMessagePayloadSchema,
  encodePacket,
  Packet,
  PacketType,
  ReadMessagePayload,
  SendMessageResponsePayloadSchema,
  SendMessageRequestPayload,
  TypingEventPayload,
  TypingEventPayloadSchema,
  BroadcastMessagePayloadSchema,
  PresencePayloadSchema,
} from "@/lib/ws/proto";

type WSContext = {
  sendMessage: (roomID: string, type: MessageType, data: string) => void;
  readMessage: (roomID: string, messageID?: number) => void;
  emitTypingEvent: (roomID: string, typing: boolean) => void;
  typing: Record<string, string[]>;
  online: Record<string, boolean>;
};

const wsContext = createContext<WSContext>({
  sendMessage: () => {},
  readMessage: () => {},
  emitTypingEvent: () => {},
  typing: {},
  online: {},
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
  const [online, setOnline] = useState<Record<string, boolean>>({});
  const { mutate } = useSWRConfig();
  const session = useSession();

  const emitTypingEvent = (roomID: string, typing: boolean) => {
    const payload: TypingEventPayload = {
      roomID: roomID,
      typing: typing,
      username: session.username,
    };

    const packet = encodePacket(
      PacketType.TypingEventPacket,
      JSON.stringify(payload)
    );
    ws.current.sendPacket(packet);
  };

  const handleBroadcastReadMessagePacket = (packet: Packet) => {
    const payloadValidation = BroadcastReadMessagePayloadSchema.safeParse(
      packet.payload
    );

    if (!payloadValidation.success) {
      throw new Error(
        "invalid BroadcastReadMessagePayload format: " +
          JSON.stringify(payloadValidation.error.format())
      );
    }

    const payload = payloadValidation.data;
    mutate(
      `/api/rooms/${payload.roomID}/messages`,
      async (currentMessages: Message[] | undefined) => {
        if (!currentMessages) return [];
        return currentMessages.map((message) => {
          if (
            message.id <= payload.messageID &&
            !message.interactions.find(
              (i) => i.username === payload.username
            ) &&
            message.sender != payload.username
          ) {
            return {
              ...message,
              interactions: [
                ...message.interactions,
                {
                  username: payload.username,
                  readAt: payload.readAt,

                  messageID: message.id,
                },
              ],
            };
          }
          return message;
        });
      },
      { revalidate: false }
    );
  };

  const readMessage = useCallback(
    (roomID: string, messageID?: number) => {
      const payload: ReadMessagePayload = {
        roomID: roomID,
      };
      const packet = encodePacket(
        PacketType.ReadMessagePacket,
        JSON.stringify(payload)
      );

      ws.current.sendPacket(packet);
      if (!messageID) return;

      mutate(
        `/api/rooms/${roomID}`,
        async (room: Room | undefined) => {
          if (!room) return;
          return {
            ...room,
            users: room.users.map((u) => {
              if (u.username === session.username) {
                return {
                  ...u,
                  lastMessageRead: messageID,
                };
              }
              return u;
            }),
          };
        },
        { revalidate: false }
      );
    },
    [ws.current, mutate]
  );

  const handleBroadcastMessagePacket = (packet: Packet) => {
    const payloadValidation = BroadcastMessagePayloadSchema.safeParse(
      packet.payload
    );

    if (!payloadValidation.success) {
      throw new Error(
        "invalid BroadcastMessagePayload format: " +
          JSON.stringify(payloadValidation.error.format())
      );
    }

    const payload = payloadValidation.data;

    mutate(
      `/api/rooms/${payload.roomID}/messages`,
      async (currentMessages: Message[] | undefined) => {
        if (!currentMessages) return [];
        return [payload, ...currentMessages];
      },
      { revalidate: false }
    );
  };

  const handleSendMessageResponsePacket = (packet: Packet) => {
    const payloadValidation = SendMessageResponsePayloadSchema.safeParse(
      packet.payload
    );

    if (!payloadValidation.success) {
      throw new Error(
        "invalid SendMessageResponsePayload format: " +
          JSON.stringify(payloadValidation.error.format())
      );
    }

    const payload = payloadValidation.data;

    mutate(
      `/api/rooms/${payload.roomID}/messages`,
      async (currentMessages: Message[] | undefined) => {
        if (!currentMessages) return [];
        return currentMessages.map((message) => {
          if (
            typeof message.correlationID === "number" &&
            message.correlationID === packet.correlationID
          ) {
            return {
              ...message,
              id: payload.messageID,
              sentAt: payload.sentAt,
              correlationID: undefined,
            };
          }
          return message;
        });
      },
      { revalidate: false }
    );

    mutate(
      `/api/rooms/${payload.roomID}`,
      async (room: Room | undefined) => {
        if (!room) return;
        return {
          ...room,
          users: room.users.map((u) => {
            if (u.username === session.username) {
              return {
                ...u,
                lastMessageRead: payload.messageID,
              };
            }
            return u;
          }),
        };
      },
      { revalidate: false }
    );
  };

  const sendMessage = (roomID: string, type: MessageType, data: string) => {
    const payload: SendMessageRequestPayload = {
      roomID: roomID,
      type: type,
      data: data,
    };

    const packet = encodePacket(
      PacketType.SendMessageRequestPacket,
      JSON.stringify(payload)
    );

    ws.current.sendPacket(packet);

    const newMessage: Message = {
      id: 0,
      correlationID: packet.correlationID,
      data: data,
      type: type,
      sender: session.username,
      sentAt: new Date().toISOString(),
      roomID: roomID,
      interactions: [],
    };

    mutate(
      `/api/rooms/${roomID}/messages`,
      async (currentMessages: Message[] | undefined) => [
        newMessage,
        ...(currentMessages ? currentMessages : []),
      ],
      {
        revalidate: false,
      }
    );
  };

  const handleTypingEventPacket = (packet: Packet) => {
    const payloadValidartion = TypingEventPayloadSchema.safeParse(
      packet.payload
    );

    if (!payloadValidartion.success) {
      throw new Error(
        "invalid TypingEventPayload format: " +
          JSON.stringify(payloadValidartion.error.format())
      );
    }

    const { typing, username, roomID } = payloadValidartion.data;
    if (typing) {
      setTyping((prev) => {
        if (prev[roomID]) {
          const users = prev[roomID];
          return {
            ...prev,
            [roomID]: [...users, username],
          };
        } else {
          return {
            ...prev,
            [roomID]: [username],
          };
        }
      });
    } else {
      setTyping((prev) => {
        if (prev[roomID]) {
          const users = prev[roomID];
          return {
            ...prev,
            [roomID]: users.filter((u) => u !== username),
          };
        } else {
          return prev;
        }
      });
    }
  };

  const handlePresencePacket = (packet: Packet) => {
    const payloadValidation = PresencePayloadSchema.safeParse(packet.payload);
    if (!payloadValidation.success) {
      throw new Error(
        "invalid PresencePayload format: " +
          JSON.stringify(payloadValidation.error.format())
      );
    }
    const payload = payloadValidation.data;
    setOnline((prev) => ({ ...prev, [payload.username]: payload.presence }));
  };

  const handlePacketReceived = (packet: Packet) => {
    try {
      switch (packet.type) {
        case PacketType.SendMessageResponsePacket:
          handleSendMessageResponsePacket(packet);
          break;

        case PacketType.BroadcastMessagePacket:
          handleBroadcastMessagePacket(packet);
          break;

        case PacketType.BroadcastReadMessagePacket:
          handleBroadcastReadMessagePacket(packet);
          break;
        case PacketType.TypingEventPacket:
          handleTypingEventPacket(packet);
          break;

        case PacketType.PresencePacket:
          handlePresencePacket(packet);
          break;
        default:
          console.log("Cannot handle packet", packet);
      }
    } catch (e) {
      console.error(e);
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
      <div className="w-screen h-screen flex flex-col gap-4 items-center justify-center">
        <p>You are offline</p>
        <Button onClick={() => ws.current.connect()}>Reconnect</Button>
      </div>
    );
  }

  return (
    <wsContext.Provider
      value={{
        sendMessage,
        readMessage,
        emitTypingEvent,
        typing,
        online,
      }}
    >
      <Outlet />
    </wsContext.Provider>
  );
}

export const useTyping = (roomID: string) => {
  const { typing } = useWS();
  return typing[roomID] || [];
};

export const usePresence = (usernames: string[]) => {
  const { online: presence } = useWS();
  return usernames.map((username) => presence[username]);
};
