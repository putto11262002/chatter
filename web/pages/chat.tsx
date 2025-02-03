import { useMyRooms } from "@/hooks/chats";
import { Link, useParams } from "react-router-dom";
import Alert from "@/components/alert";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import ChatArea from "@/components/chat/chat-area";
import { Button } from "@/components/ui/button";
import { LogOut, MessageCirclePlus } from "lucide-react";
import { useCreateRoomDialog } from "@/components/create-room-dialog";
import { useSignout } from "@/hooks/auth";

export default function ChatPage() {
  const { setOpen } = useCreateRoomDialog();
  const { trigger: signout, isMutating: isSigningOut } = useSignout();
  const { data, isLoading, error } = useMyRooms();
  const params = useParams();
  const roomID = params.roomID;

  return (
    <main className="h-screen w-full">
      <div className="grid grid-cols-[20%_80%] h-screen overflow-hidden">
        <div className="h-full flex flex-col border-r">
          <div className="flex-0 py-2 px-2 border-b">
            <Button onClick={() => setOpen(true)} variant="outline" size="icon">
              <MessageCirclePlus className="w-6 h-6" />
            </Button>
          </div>
          <div className="grow">
            {error ? (
              <Alert message={error.message} />
            ) : isLoading || !data ? (
              <div className="grid gap-1">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : (
              <div className="grid">
                {data.map((room, index) => {
                  return (
                    <Link key={index} to={`/${room.id}`}>
                      <div
                        key={index}
                        className={cn(
                          "h-12 px-3 border-b hover:bg-gray-50 cursor-pointer flex gap-3 items-center",
                          roomID === room.id && "bg-gray-100"
                        )}
                      >
                        <p className="font-medium grow">{room.name}</p>
                      </div>
                    </Link>
                  );
                })}
              </div>
            )}
          </div>
          <div className="flex-0 py-2 px-2 border-t h-14">
            <Button
              variant="outline"
              size="icon"
              disabled={isSigningOut}
              onClick={() => signout()}
            >
              <LogOut className="w-6 h-6" />
            </Button>
          </div>
        </div>
        <div className="h-full overflow-hidden">
          <ChatArea />
        </div>
      </div>
    </main>
  );
}
