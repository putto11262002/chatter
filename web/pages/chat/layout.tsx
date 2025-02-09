import { Outlet } from "react-router-dom";
import { CreateRoomDialogProvider } from "./components/create-room-dialog";

export default function ChatLayout() {
  return (
    <CreateRoomDialogProvider>
      <Outlet />
    </CreateRoomDialogProvider>
  );
}
