import { useEffect, useRef } from "react";
import React from "react";
import { Packet, ReadyState, WS } from "./ws";

type ChatContext = {
  readyState: ReadyState;
  ws: WS;
};

const wsContext = React.createContext<ChatContext>({
  ws: new WS({ onStateChange: () => {}, onPacketReceived: () => {} }),
  readyState: ReadyState.Closed,
});

export const useWS = () => React.useContext(wsContext);

export type EventHandler = (e: Packet) => void;

export function WSProvider({
  children,
  handlers,
}: {
  children: React.ReactNode;
  handlers: Record<string, EventHandler>;
}) {
  const [readyState, setReadyState] = React.useState<ReadyState>(
    ReadyState.Closed
  );

  const ws = useRef<WS>(
    new WS({
      onStateChange: setReadyState,
      onPacketReceived: handlePacket,
    })
  );

  function handlePacket(packet: Packet) {
    const handler = handlers[packet.type];
    if (!handler) {
      console.error("No handler for packet type", packet);
      return;
    }
    console.log("Handling packet", packet);
    handler(packet);
  }

  useEffect(() => {
    ws.current.connect();
  }, []);

  return (
    <wsContext.Provider value={{ readyState, ws: ws.current }}>
      {children}
    </wsContext.Provider>
  );
}
