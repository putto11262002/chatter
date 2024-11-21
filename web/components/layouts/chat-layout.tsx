import CreatePrivateChatDialog from "@/components//create-private-chat-dialog";
import { useMyRooms } from "@/hooks/chats";
import { Link, Outlet, useParams } from "react-router-dom";
import Alert from "../alert";
import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import Avatar from "../avatar";

export default function ChatLayout() {
  const { data, isLoading, error } = useMyRooms();
  const params = useParams();
  const roomID = params.roomID;

  return (
    <main className="h-screen w-full">
      <div className="grid grid-cols-[30%_70%] h-screen overflow-hidden">
        <div className="h-full flex flex-col border-r">
          <div className="flex-0 py-2 px-2 border-b">
            <CreatePrivateChatDialog />
          </div>
          <div className="grow">
            {error ? (
              <Alert message={error.message} />
            ) : isLoading || !data ? (
              <div>
                <Loader2 className="w-4 h-4 animate-spin" />
              </div>
            ) : (
              <div className="grid">
                {data.map((room, index) => (
                  <Link to={`/${room.roomID}`}>
                    <div
                      key={index}
                      className={cn(
                        "py-2 px-3 border-b hover:bg-accent cursor-pointer flex gap-3",
                        roomID === room.roomID && "bg-accent"
                      )}
                    >
                      <Avatar name={room.roomName} />
                      <p className="font-medium grow">{room.roomName}</p>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
        <div className="h-full overflow-hidden">
          <Outlet />
        </div>
      </div>
    </main>
  );
}
