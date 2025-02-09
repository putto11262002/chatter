import RoomMemberForm from "@/pages/room-settings/room-member-form";
import RoomProfileForm from "./room-profile-form";
import { Button } from "@/components/ui/button";
import { useRoom } from "@/hooks/react-query/chats"
import { ChevronLeft, Loader2 } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";

export default function RoomSettingsPage() {
  const params = useParams();
  const roomID = params.roomID;
  const { data: room } = useRoom(roomID);
  const nagivate = useNavigate();

  if (!room) {
    return (
      <main className="flex justify-center py-4">
        <Loader2 className="w-4 h-4 animate-spin" />
      </main>
    );
  }

  return (
    <main className="flex justify-center">
      <div className="container py-4 px-4 grid gap-4">
        <div className="flex items-center gap-4">
          <Button
            onClick={() => nagivate(-1)}
            variant="outline"
            size="icon"
            className="w-7 h-7"
          >
            <ChevronLeft className="w-4 h-4" />
          </Button>
          <h1 className="text-2xl font-bold">{room.name} Settings</h1>
        </div>
        <div className="grid gap-6">
          <div className="grid gap-2">
            <h2 className="text-xl font-bold">Room Profile</h2>
            <RoomProfileForm room={room} />
          </div>
          <div className="grid gap-2">
            <h2 className="text-xl font-bold">Members</h2>
            <RoomMemberForm room={room} />
          </div>
        </div>
      </div>
    </main>
  );
}
