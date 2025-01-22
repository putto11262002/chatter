import { Room } from "@/lib/types/chat";
import { RoomMemberContextProvider } from "./context";
import Form from "./form";
import AddMemberDialog from "./dialog";

export default function RoomMemberForm({ room }: { room: Room }) {
  return (
    <RoomMemberContextProvider>
      <Form room={room} />
      <AddMemberDialog room={room} />
    </RoomMemberContextProvider>
  );
}
