import { useEffect, useRef, useLayoutEffect, useMemo } from "react";
import { useInfiniteMessages, useRoom } from "@/hooks/react-query/chats";
import { cn } from "@/lib/utils";
import { differenceInMinutes, format, isSameDay } from "date-fns";
import Avatar from "@/components/avatar";
import { LoaderIcon } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useRealtimeStore } from "@/stores/real-time";
import { useMessageScroll } from "./message-scroll-context";
import { UserRealtimeInfo } from "@/stores/user";
import { useSession } from "@/context/session";
import Message from "./message";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  const {
    data,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    isInitialLoading,
  } = useInfiniteMessages(roomID);
  const { data: room } = useRoom(roomID);

  const realtimeMessages =
    useRealtimeStore((state) => state.messages)[roomID] || [];
  const realtimeUserInfo = useRealtimeStore((state) => state.users);
  const roomMemberInfo = useMemo(
    () =>
      room?.members.reduce<Array<UserRealtimeInfo>>((acc, member) => {
        const memberInfo = realtimeUserInfo[member.username];
        if (memberInfo) {
          acc.push(memberInfo);
        }
        return acc;
      }, []),
    [room, realtimeUserInfo]
  );
  const typingMembers = roomMemberInfo
    ?.filter((member) => member.username !== session.username)
    .filter((member) => member.typing === roomID);

  const messages = [...(data?.pages || []), realtimeMessages];

  const { ref: scrollAreaRef, getScrollContainer } = useMessageScroll();
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
        const scrollContainer = getScrollContainer();
        if (
          entry.isIntersecting &&
          hasNextPage &&
          scrollContainer &&
          !isFetchingNextPage
        ) {
          // Store both scroll height and position before loading more messages"
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
    const scrollContainer = getScrollContainer();
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

  useLayoutEffect(() => {
    const lastest = realtimeMessages
      ? realtimeMessages[realtimeMessages.length - 1]
      : null;
    if (!lastest) return;
    // if you sent the lastest message scroll to the bottom
    const scrollContainer = getScrollContainer();
    if (!scrollContainer) return;
    scrollContainer.scrollTop = scrollContainer.scrollHeight;
  }, [realtimeMessages]);

  useLayoutEffect(() => {
    if (!typingMembers) return;
    if (typingMembers.length === 0) return;
    const scrollContainer = getScrollContainer();
    if (!scrollContainer) return;
    // if not already at the bottom scroll to the bottom
    if (scrollContainer.scrollTop !== scrollContainer.scrollHeight) {
      scrollContainer.scrollTop = scrollContainer.scrollHeight;
    }
  }, [typingMembers]);

  return (
    <ScrollArea
      ref={scrollAreaRef}
      className="h-full relative w-full [&>div>div[style]]:!block"
    >
      <div className="flex flex-col gap-2 px-4 py-2 w-full min-w-0 overflow-hidden ">
        <div
          className="flex justify-center items-center w-full min-w-0"
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
        {messages.flat().map((message, index, messages) => {
          const myMessage = message.sender === session.username;
          const nextMsg =
            index + 1 <= messages.length - 1 ? messages[index + 1] : null;
          const prevMsg = index - 1 >= 0 ? messages[index - 1] : null;
          const newDay =
            prevMsg && !isSameDay(prevMsg.sent_at, message.sent_at);
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
              className={cn(
                "flMex flex-col overflow-hidden min-w-0 w-full",
                endOfGroup && "mb-2"
              )}
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
                  "flex flex-col overflow-hidden w-full",
                  myMessage ? "items-end" : "items-start"
                )}
              >
                <div className={cn("flex items-end gap-2 max-w-[70%] min-w-0")}>
                  {!myMessage && (
                    <div className="shrink-0 w-7">
                      {shouldDisplaySender && (
                        <Avatar
                          online={realtimeUserInfo[message.sender]?.online}
                          size="sm"
                          name={message.sender}
                        />
                      )}
                    </div>
                  )}
                  <Message
                    message={message}
                    className={cn(!myMessage && "bg-accent")}
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
        {typingMembers && typingMembers.length > 0 && (
          <div className="flex items-center gap-2">
            {typingMembers.slice(0, 4).map((member) => (
              <div
                key={member.username}
                className="animate-bounce [animation-duration:300ms]"
              >
                <Avatar size="sm" name={member.username} />
              </div>
            ))}
          </div>
        )}
      </div>
    </ScrollArea>
  );
}
