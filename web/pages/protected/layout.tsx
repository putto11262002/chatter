import { SessionProvider } from "@/context/session";
import { WSProvider } from "@/context/ws";
import { eventHandlers } from "@/ws/handlers";
import { Outlet } from "react-router-dom";

export default function ProtectedLayut() {
  return (
    <SessionProvider>
      <WSProvider handlers={eventHandlers}>
        <Outlet />
      </WSProvider>
    </SessionProvider>
  );
}
