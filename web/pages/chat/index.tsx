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
import { useWS } from "@/real-time/context";
import { ReadyState } from "@/real-time/ws";
import { useRealtimeStore } from "@/store/real-time";
import { formatDistance } from "date-fns";
import { ScrollArea } from "@/components/ui/scroll-area";
import { RoomList } from "./rooms/list";
import { useEffect, useState } from "react";
import resolveConfig from "tailwindcss/resolveConfig";

const useMediaQuery = (query: string) => {
  const mediaQuery = window.matchMedia(query);
  const [matches, setMatches] = useState(mediaQuery.matches);

  useEffect(() => {
    const handler = (e: MediaQueryListEvent) => setMatches(e.matches);
    mediaQuery.addEventListener("change", handler);
    return () => mediaQuery.removeEventListener("change", handler);
  }, []);

  return matches;
};

export default function ChatPage() {
  const { setOpen } = useCreateRoomDialog();
  const { trigger: signout, isMutating: isSigningOut } = useSignout();
  const params = useParams();
  const roomID = params.roomID;
  const { readyState } = useWS();
  const isSmallScreen = useMediaQuery(`(max-width: 1024px)`);

  return (
    <main className="h-screen w-full">
      <div className="grid lg:grid-cols-[20%_80%] h-screen ">
        {(isSmallScreen ? !roomID : true) && (
          <div className="h-full flex flex-col border-r overflow-hidden">
            <div className="flex-0 py-2 px-2 border-b flex justify-between items-center">
              <Button
                onClick={() => setOpen(true)}
                variant="outline"
                size="icon"
              >
                <MessageCirclePlus className="w-6 h-6" />
              </Button>
              <div className="px-2 ">
                <div
                  className={cn(
                    "rounded-full h-4 w-4 text-xs font-medium",
                    readyState === ReadyState.Open &&
                      "bg-green-300 border-green-500 text-green-900"
                  )}
                ></div>
              </div>
            </div>

            <div className="grow overflow-hidden">
              <RoomList />
            </div>
            <div className="flex-0 py-2 px-2 border-t h-14 flex items-center">
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
        )}
        {(isSmallScreen ? roomID : true) && (
          <div className="h-full overflow-hidden">
            <ChatArea />
          </div>
        )}
      </div>
    </main>
  );
}
