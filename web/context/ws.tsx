import { useEffect, useRef } from "react";
import React from "react";
import { EventHandler, ReadyState, WS } from "../lib/ws";

type ChatContext = {
  readyState: ReadyState;
  ws: WS;
};

const wsContext = React.createContext<ChatContext>({
  ws: new WS({ onStateChange: () => {}, handlers: {} }),
  readyState: ReadyState.Closed,
});

export const useWS = () => React.useContext(wsContext);

export function WSProvider({
  children,
  handlers,
  onReadyStateChange,
}: {
  children: React.ReactNode;
  handlers: Record<string, EventHandler>;
  onReadyStateChange?: (state: ReadyState) => void;
}) {
  const [readyState, setReadyState] = React.useState<ReadyState>(
    ReadyState.Closed
  );

  const ws = useRef<WS>(
    new WS({
      onStateChange: setReadyState,
      handlers: handlers,
    })
  );

  useEffect(() => {
    if (onReadyStateChange) onReadyStateChange(readyState);
  }, [readyState, onReadyStateChange]);

  useEffect(() => {
    ws.current.connect();
  }, []);

  return (
    <wsContext.Provider value={{ readyState, ws: ws.current }}>
      {children}
    </wsContext.Provider>
  );
}
