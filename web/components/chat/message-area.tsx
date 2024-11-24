import { useChatMessageHistory, useRoom } from "@/hooks/chats";
import { useSession } from "../providers/session-provider";
import Alert from "../alert";
import { Loader2, MoreHorizontal } from "lucide-react";
import Message from "./message";
import { cn } from "@/lib/utils";
import { useEffect, useRef, useState } from "react";
import { ScrollArea } from "../ui/scroll-area";
import { useTyping, useWS } from "@/hooks/ws-provider";
import { differenceInDays, format } from "date-fns";
import Avatar from "../avatar";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  const { data, isLoading, error } = useChatMessageHistory(roomID);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const messages = data?.slice().reverse();
  const { online } = useWS();
  const { data: room, isLoading: isLoadingRoom } = useRoom(roomID);
  const { readMessage } = useWS();
  const users = useTyping(roomID);
  const [tabActive, setTabActive] = useState(false);

  useEffect(() => {
    window.addEventListener("focus", () => {
      setTabActive(true);
    });
    window.addEventListener("blur", () => {
      setTabActive(false);
    });

    return () => {
      window.removeEventListener("focus", () => {});
      window.removeEventListener("blur", () => {});
      setTabActive(false);
    };
  }, []);

  useEffect(() => {
    if (data && !isLoading && data.length > 0) {
      readMessage(roomID, data[0].id);
    }
  }, [roomID, data, isLoading]);

  useEffect(() => {
    if (!scrollAreaRef.current) return;

    // Allow the DOM to update before scrolling
    requestAnimationFrame(() => {
      const scrollContainer = scrollAreaRef.current?.querySelector(
        "[data-radix-scroll-area-viewport]"
      );
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    });
  }, [messages]);

  if (error) {
    return (
      <div>
        <Alert message={error.message} />
      </div>
    );
  }

  if (isLoading || !messages || isLoadingRoom || !room) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-4 h-4 animate-spin" />
      </div>
    );
  }

  return (
    <ScrollArea ref={scrollAreaRef} className="h-full relative">
      <div className="flex flex-col gap-2 px-4 py-2">
        {messages.map((message, index) => {
          const myMessage = message.sender === session.username;
          const next =
            index + 1 <= messages.length - 1 ? messages[index + 1] : null;

          const prev = index - 1 >= 0 ? messages[index - 1] : null;

          const nextRead = next && next.interactions.length > 0;

          const read = message.interactions.length > 0;

          const readUpToHere = room?.users
            .filter((u) => u.lastMessageRead === message.id)
            .map((u) => u.username);

          return (
            <div key={index} className="flex flex-col">
              {prev &&
                Math.abs(differenceInDays(prev.sentAt, message.sentAt)) > 1 && (
                  <p className="text-center text-sm text-muted-foreground mt-2 pb-1">
                    {format(message.sentAt, "dd/MM/yyyy")}
                  </p>
                )}
              <div className="flex gap-2">
                {!myMessage && (
                  <Avatar
                    online={online[message.sender]}
                    size="sm"
                    name={message.sender}
                  />
                )}
                <div
                  className={cn(
                    "flex flex-col grow gap-1",
                    myMessage ? "items-end" : "items-start"
                  )}
                >
                  <div className="max-w-[70%]">
                    <Message
                      message={message}
                      className={myMessage ? "bg-gray-200" : ""}
                    />
                  </div>
                </div>
              </div>
              {readUpToHere && readUpToHere.length > 0 && (
                <div className="flex items-center gap-3 mt-2 pb-2">
                  <div className="flex items-center gap-1">
                    {readUpToHere.map((username, index) => (
                      <Avatar
                        key={index}
                        name={username}
                        size="xs"
                        online={online[username]}
                      />
                    ))}
                  </div>
                  <div className="grow border-b border-dashed"></div>
                </div>
              )}
            </div>
          );
        })}
        {users.length > 0 && (
          <div className="flex items-start">
            <div
              className={cn(
                "px-4 py-2 rounded-lg border animate-in animate-out"
              )}
            >
              <MoreHorizontal className="w-4 h-4 animate-ping" />
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  );
}
