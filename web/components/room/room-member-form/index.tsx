import { Room } from "@/lib/types/chat";
import { RoomMemberContextProvider } from "./context";
import Table from "./table";
import AddMemberDialog from "./dialog";

export default function RoomMemberForm({ room }: { room: Room }) {
  return (
    <RoomMemberContextProvider>
      <Table room={room} />
      <AddMemberDialog room={room} />
    </RoomMemberContextProvider>
  );
}
