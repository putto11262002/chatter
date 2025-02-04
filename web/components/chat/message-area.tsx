import { useState, useEffect, useRef, useLayoutEffect } from "react";
import { useInfiniteScrollMessageHistory, useRoom } from "@/hooks/chats";
import { useSession } from "../providers/session-provider";
import Alert from "../alert";
import { Loader2 } from "lucide-react";
import Message from "./message";
import { cn } from "@/lib/utils";
import { differenceInDays, differenceInMinutes, format } from "date-fns";
import Avatar from "../avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { ScrollArea } from "../ui/scroll-area";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  const {
    data: pages,
    isLoading,
    size,
    setSize,
    error,
  } = useInfiniteScrollMessageHistory(roomID);

  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const { data: room, isLoading: isLoadingRoom } = useRoom(roomID);
  const oldestMessageRef = useRef<HTMLDivElement>(null);
  const messages = pages?.reverse().flat() || [];
  const hasMore = pages ? pages[pages.length - 1].length === 20 : false;

  // Ref to store the previous scroll height before fetching new messages
  const previousScrollHeightRef = useRef<number>(0);

  // Intersection observer to load more messages when the oldest message is in view
  useEffect(() => {
    const observer = new IntersectionObserver(
      async (entries) => {
        const entry = entries[0];
        if (entry.isIntersecting && hasMore) {
          setSize((size) => size + 1);
          if (scrollAreaRef.current) {
            if (scrollAreaRef.current) {
              // Record scroll height before fetching older messages

              previousScrollHeightRef.current =
                scrollAreaRef.current.scrollHeight -
                scrollAreaRef.current.scrollTop;
              console.log(
                "scrollHeight",
                scrollAreaRef.current.scrollHeight,
                "from observer previousScrollHeightRef.current",
                previousScrollHeightRef.current
              );
            }
          }
        }
      },
      {
        threshold: 1,
        root:
          scrollAreaRef.current?.querySelector(
            "[data-radix-scroll-area-viewport]"
          ) || null,
      }
    );
    if (oldestMessageRef.current) observer.observe(oldestMessageRef.current);
    return () => observer.disconnect();
  }, [pages, isLoading, hasMore]);

  // Adjust scroll position after older messages are loaded
  useLayoutEffect(() => {
    if (!(previousScrollHeightRef.current || scrollAreaRef.current)) return;

    requestAnimationFrame(() => {
      const newScrollHeight = scrollAreaRef.current.scrollHeight;
      const heightDiff = newScrollHeight - previousScrollHeightRef.current;
      scrollAreaRef.current.scrollTop = heightDiff;
    });
  }, [pages]);

  return (
    <div ref={scrollAreaRef} className="h-full overflow-y-auto">
      <div className="flex flex-col gap-2 px-4">
        <div ref={oldestMessageRef}></div>
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
          const endOfGroup = lastFromSameSender || lastOfTheMinute;
          const shouldDisplaySender = !myMessage && endOfGroup;

          return (
            <div
              key={message.id}
              className={cn("flex flex-col", endOfGroup && "mb-2")}
            >
              {newDay && (
                <div className="mt-2 pb-1 pt-2 border-t">
                  <p className="text-center text-sm text-muted-foreground">
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
                <div className={cn("flex items-end gap-2 max-w-[70%]")}>
                  {!myMessage && (
                    <div className="shrink-0 w-7">
                      {shouldDisplaySender && (
                        <Avatar size="sm" name={message.sender} />
                      )}
                    </div>
                  )}
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
                {endOfGroup && (
                  <p
                    className={cn(
                      "text-xs text-muted-foreground mt-2",
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
    </div>
  );
}
