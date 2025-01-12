import { useRoom } from "@/hooks/chats";
import { Skeleton } from "../ui/skeleton";
import { Button } from "../ui/button";
import { MoreHorizontal } from "lucide-react";

export default function ChatHeader({ roomID }: { roomID: string }) {
  const { isLoading, data: room } = useRoom(roomID);
  if (!roomID) {
    return null;
  }

  return (
    <div className="flex px-4 py-2 w-full border-b items-center">
      {isLoading || !room ? (
        <Skeleton className="w-1/3 h-9" />
      ) : (
        <>
          <h1 className="grow">{room.id}</h1>
          <Button size="icon" variant="outline">
            <MoreHorizontal className="w-4 h-4" />
          </Button>
        </>
      )}
    </div>
  );
}
