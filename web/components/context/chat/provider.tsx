import { Packet, ReadyState, WS } from "@/lib/ws";
import { MessageBody, messageBodySchema, PacketType } from "@/lib/ws/data";
import { useEffect, useRef } from "react";
import { useSWRConfig } from "swr";
import React from "react";
import { Message } from "@/lib/types/chat";

type SWRConfig = ReturnType<typeof useSWRConfig>;

const handlers: Record<string, (swrConfig: SWRConfig, packet: Packet) => void> =
  {
    [PacketType.Message]: ({ mutate }: SWRConfig, packet: Packet) => {
      const result = messageBodySchema.safeParse(packet.payload);
      if (!result.success) {
        console.error("Invalid message packet received", packet);
        return;
      }
      const message = result.data;
      mutate(
        `/rooms/${message.room_id}/messages`,
        (messages: Message[] | undefined) => {
          if (!messages) {
            return [message];
          }
          return [...messages, message];
        },
        { revalidate: false }
      );

      console.log("Message packet received", packet);
    },
  };

type ChatContext = {
  sendMessage: (
    message: Pick<MessageBody, "data" | "room_id" | "type">
  ) => void;
  readyState: ReadyState;
};

const chatContext = React.createContext<ChatContext>({
  sendMessage: () => {},
  readyState: ReadyState.Closed,
});

export const useChat = () => React.useContext(chatContext);

export function ChatProvider({ children }: { children: React.ReactNode }) {
  const swrConfig = useSWRConfig();
  const [readyState, setReadyState] = React.useState<ReadyState>(
    ReadyState.Closed
  );
  const ws = useRef<WS>(
    new WS({
      onStateChange: setReadyState,
      onPacketReceived: handlePacket,
    })
  );

  function sendMessage(
    message: Pick<MessageBody, "data" | "room_id" | "type">
  ) {
    if (ws.current.readyState !== ReadyState.Open) {
      return;
    }
    ws.current.sendPacket({
      type: PacketType.Message,
      payload: message,
    });
  }

  function handlePacket(packet: Packet) {
    const handler = handlers[packet.type];
    if (handler) {
      handler(swrConfig, packet);
    }
  }

  useEffect(() => {
    ws.current.connect();
    console.log("Connecting to ws");
  }, []);

  return (
    <chatContext.Provider value={{ sendMessage, readyState }}>
      {children}
    </chatContext.Provider>
  );
}
