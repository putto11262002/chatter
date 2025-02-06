import { Room } from "@/lib/types/chat";
import { cn } from "@/lib/utils";
import { formatDistance } from "date-fns";
import { useParams } from "react-router-dom";

const RoomListItem = ({ room }: { room: Room }) => {
  const params = useParams();
  const roomInView = params.roomID;
  return (
    <div
      className={cn(
        "w-full px-3 h-16 border-b hover:bg-accent cursor-pointer flex flex-col justify-center",
        roomInView === room.id && "bg-accent"
      )}
    >
      <p className="font-medium">{room.name}</p>
      {/* Display last sent message if exist  */}
      {room.last_message_sent !== 0 && (
        <div className="flex items-center gap-1">
          <p className="text-xs text-muted-foreground grow max-w-2/3 overflow-hidden text-ellipsis whitespace-nowrap">
            {room.last_message_sent_data}
          </p>
          <p className="whitespace-nowrap text-xs">
            {formatDistance(room.last_message_sent_at, new Date(), {
              addSuffix: true,
            })}
          </p>
        </div>
      )}
    </div>
  );
};
export default RoomListItem;
