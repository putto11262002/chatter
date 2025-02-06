import { Button } from "../ui/button";
import { ChevronLeft, MoreHorizontal } from "lucide-react";
import { Link } from "react-router-dom";
import { Room } from "@/lib/types/chat";

export default function ChatHeader({ room }: { room: Room }) {
  return (
    <div className="flex px-4 py-2 w-full border-b items-center gap-3">
      <Link className="lg:hidden" to="/">
        <Button size="icon" variant="outline">
          <ChevronLeft className="w-4 h-4" />
        </Button>
      </Link>

      <h1 className="grow">{room.name}</h1>
      <Link to={`/rooms/${room.id}/settings`}>
        <Button size="icon" variant="outline">
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </Link>
    </div>
  );
}
