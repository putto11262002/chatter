import { useEffect, useRef, useLayoutEffect } from "react";
import { useInfiniteMessages } from "@/hooks/chats";
import { useSession } from "../providers/session-provider";
import Message from "./message";
import { cn } from "@/lib/utils";
import { differenceInDays, differenceInMinutes, format } from "date-fns";
import Avatar from "../avatar";
import { LoaderIcon } from "lucide-react";
import { ScrollArea } from "../ui/scroll-area";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  // const {
  //   data: pages,
  //   isLoading,
  //   setSize,
  // } = useInfiniteScrollMessageHistory(roomID);
  const {
    data,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    isInitialLoading,
  } = useInfiniteMessages(roomID);

  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const oldestMessageRef = useRef<HTMLDivElement>(null);

  // Store both the scroll height and scroll position
  const scrollPositionRef = useRef<{
    scrollHeight: number;
    scrollTop: number;
  }>({ scrollHeight: 0, scrollTop: 0 });

  // Intersection observer to load more messages when the oldest message is in view
  useEffect(() => {
    const observer = new IntersectionObserver(
      async (entries) => {
        const entry = entries[0];
        if (
          entry.isIntersecting &&
          hasNextPage &&
          scrollAreaRef.current &&
          !isFetchingNextPage
        ) {
          const scrollContainer = scrollAreaRef.current?.querySelector(
            "[data-radix-scroll-area-viewport]"
          );
          if (!scrollContainer) return;
          // Store both scroll height and position before loading more messages
          scrollPositionRef.current = {
            scrollHeight: scrollContainer.scrollHeight,
            scrollTop: scrollContainer.scrollTop,
          };

          fetchNextPage();
        }
      },
      {
        threshold: 0.1,
        root: scrollAreaRef.current || null,
      }
    );

    if (oldestMessageRef.current) observer.observe(oldestMessageRef.current);
    return () => observer.disconnect();
  }, [data, hasNextPage, isFetchingNextPage]);

  // Use useLayoutEffect to adjust scroll position before browser paint
  useLayoutEffect(() => {
    if (!scrollAreaRef.current) return;
    const scrollContainer = scrollAreaRef.current?.querySelector(
      "[data-radix-scroll-area-viewport]"
    );
    if (!scrollContainer) return;

    const { scrollHeight: previousScrollHeight, scrollTop: previousScrollTop } =
      scrollPositionRef.current;

    const newScrollHeight = scrollContainer.scrollHeight;
    const newScrollTop =
      newScrollHeight - previousScrollHeight + previousScrollTop;

    // Immediately set the scroll position to maintain the relative view
    scrollContainer.scrollTop = newScrollTop;

    // // Reset the stored position
    scrollPositionRef.current = { scrollHeight: 0, scrollTop: 0 };
  }, [data]);

  return (
    <ScrollArea ref={scrollAreaRef} className="h-full relative">
      <div className="flex flex-col gap-2 px-4 py-2">
        <div
          className="flex justify-center items-center"
          ref={oldestMessageRef}
        >
          {hasNextPage || isInitialLoading ? (
            <p className="flex items-center justify-center gap-2">
              <LoaderIcon className="w-4 h-4 animate-spin" />{" "}
              <span className="text-xs text-muted-foreground">
                Fetching more messages
              </span>{" "}
            </p>
          ) : (
            <p className="text-xs text-center text-muted-foreground">
              No more messages
            </p>
          )}
        </div>
        {data?.pages.flat().map((message, index, messages) => {
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
                  <Message
                    message={message}
                    className={!myMessage ? "bg-accent" : ""}
                  />
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
    </ScrollArea>
  );
}
