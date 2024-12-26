import { WS } from "@/lib/ws";
import { useEffect, useRef } from "react";
import { Outlet } from "react-router-dom";

export function ChatProvider() {
  const ws = useRef<WS>(
    new WS({
      onStateChange: console.log,
      onPacketReceived: console.log,
    })
  );

  useEffect(() => {
    ws.current.connect();
  }, []);

  return (
    <>
      <Outlet />
    </>
  );
}
