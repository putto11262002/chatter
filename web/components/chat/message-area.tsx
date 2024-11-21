import { useChatMessageHistory } from "@/hooks/chats";
import { useSession } from "../providers/session-provider";
import Alert from "../alert";
import { Loader2, MoreHorizontal } from "lucide-react";
import Message from "./message";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";
import { ScrollArea } from "../ui/scroll-area";
import { MessageStatusLabels } from "@/types/chat";
import { useTyping } from "@/hooks/ws-provider";
import { differenceInDays, format } from "date-fns";
import Avatar from "../avatar";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  const { data, isLoading, error } = useChatMessageHistory(roomID);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const messages = data?.slice().reverse();

  const users = useTyping(roomID);

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

  if (isLoading || !messages) {
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

          return (
            <div key={index} className="flex flex-col">
              {prev &&
                Math.abs(differenceInDays(prev.sentAt, message.sentAt)) > 1 && (
                  <p className="text-center text-sm text-muted-foreground mt-2 pb-1">
                    {format(message.sentAt, "dd/MM/yyyy")}
                  </p>
                )}
              <div className="flex gap-2">
                {!myMessage && <Avatar name={message.sender} />}
                <div
                  className={cn(
                    "flex flex-col grow",
                    myMessage ? "items-end" : "items-start"
                  )}
                >
                  <div className="max-w-[70%]">
                    <Message
                      message={message}
                      className={myMessage ? "bg-gray-200" : ""}
                    />
                  </div>

                  {message.sender === session.username &&
                    (index === messages.length - 1 ||
                      (next && next.status != message.status)) && (
                      <p className="text-xs text-muted-foreground">
                        {MessageStatusLabels[message.status]}
                      </p>
                    )}
                </div>
              </div>
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
