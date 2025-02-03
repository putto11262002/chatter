import { useChatMessageHistory, useRoom } from "@/hooks/chats";
import { useSession } from "../providers/session-provider";
import Alert from "../alert";
import { Loader2 } from "lucide-react";
import Message from "./message";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";
import { ScrollArea } from "../ui/scroll-area";
import { differenceInDays, differenceInMinutes, format } from "date-fns";
import Avatar from "../avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  const { data: messages, isLoading, error } = useChatMessageHistory(roomID);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const { data: room, isLoading: isLoadingRoom } = useRoom(roomID);

  useEffect(() => {
    if (!scrollAreaRef.current) return;
    if (isLoading || !messages) return;

    // Allow the DOM to update before scrolling
    requestAnimationFrame(() => {
      const scrollContainer = scrollAreaRef.current?.querySelector(
        "[data-radix-scroll-area-viewport]"
      );
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    });
  }, [messages, isLoading]);

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
          const nextMsg =
            index + 1 <= messages.length - 1 ? messages[index + 1] : null;

          const prevMsg = index - 1 >= 0 ? messages[index - 1] : null;

          const newDay =
            prevMsg &&
            Math.abs(differenceInDays(prevMsg.sent_at, message.sent_at)) > 1;

          const lastOfTheMinute =
            !nextMsg ||
            Math.abs(differenceInMinutes(message.sent_at, nextMsg.sent_at)) > 1;

          const lastFromSameSender =
            !nextMsg || nextMsg.sender !== message.sender;

          // We define group of messages as messages that are from the same sender and are sent within 1 minutes of each other
          const endOfGroup = lastFromSameSender || lastOfTheMinute;
          const shouldDisplaySender = !myMessage && endOfGroup;

          return (
            <div
              key={index}
              className={cn("flex flex-col", endOfGroup && "mb-2")}
            >
              {newDay && (
                <div className="mt-2 pb-1 pt-2 border-t">
                  <p className="text-center text-sm text-muted-foreground ">
                    {format(message.sent_at, "dd/MM/yyyy")}
                  </p>
                </div>
              )}
              <div
                className={cn(
                  "flex flex-col",
                  myMessage ? "items-end" : "items-start"
                )}
              >
                {/* Layout to align the message to the right or left */}
                <div className={cn("flex items-end gap-2 max-w-[70%]")}>
                  {/* Sender avatar */}
                  {!myMessage && (
                    <div className="shrink-0 w-7">
                      {shouldDisplaySender && (
                        <Avatar size="sm" name={message.sender} />
                      )}
                    </div>
                  )}

                  {/* The actual message*/}
                  <Tooltip>
                    <TooltipTrigger>
                      <Message
                        message={message}
                        className={!myMessage ? "bg-accent" : ""}
                      />
                    </TooltipTrigger>
                    <TooltipContent align={myMessage ? "end" : "start"}>
                      {format(message.sent_at, "dd/MM/yyyy HH:mm")}
                    </TooltipContent>
                  </Tooltip>
                </div>

                {/* Timestamp */}
                {endOfGroup && (
                  <p
                    className={cn(
                      "text-xs text-muted-foreground  mt-2",
                      !myMessage && "ml-9"
                    )}
                  >
                    {format(message.sent_at, "dd/MM/yyyy HH:mm")}
                  </p>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </ScrollArea>
  );
}
