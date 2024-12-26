import CreatePrivateChatDialog from "@/components//create-private-chat-dialog";
import { useMyRooms, useRoom } from "@/hooks/chats";
import { Link, Outlet, useParams } from "react-router-dom";
import Alert from "../alert";
import { Circle, Dot, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import Avatar from "../avatar";
import { useWS } from "@/hooks/ws-provider";
import CreateGroupChatDialog from "../create-group-chat-dialog";

export default function ChatLayout() {
  const { data, isLoading, error } = useMyRooms();
  const params = useParams();
  const roomID = params.roomID;
  const { online } = useWS();

  return (
    <main className="h-screen w-full">
      <div className="grid grid-cols-[20%_80%] h-screen overflow-hidden">
        <div className="h-full flex flex-col border-r">
          <div className="flex-0 py-2 px-2 border-b">
            <CreatePrivateChatDialog />
            <CreateGroupChatDialog />
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
                {data.map((room, index) => {
                  const anyoneOnline = room.users.some((user) => online[user]);
                  return (
                    <Link to={`/${room.roomID}`}>
                      <div
                        key={index}
                        className={cn(
                          "py-2 px-3 border-b hover:bg-accent cursor-pointer flex gap-3 items-center",
                          roomID === room.roomID && "bg-accent",
                          anyoneOnline && "bg-green-200"
                        )}
                      >
                        <p className="font-medium grow">{room.roomName}</p>
                      </div>
                    </Link>
                  );
                })}
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
