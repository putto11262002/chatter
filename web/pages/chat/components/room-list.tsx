import { useInfiniteMyRooms } from "@/hooks/react-query/chats";
import { Link } from "react-router-dom";
import { CircleAlert, LoaderIcon } from "lucide-react";
import RoomListItem from "./room-list-item";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useEffect, useRef } from "react";
export const RoomList = () => {
  const { fetchNextPage, data, isLoading, error, hasNextPage } =
    useInfiniteMyRooms();
  const endOfListRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        if (entry.isIntersecting && hasNextPage && !isLoading) {
          fetchNextPage();
        }
      },
      {
        threshold: 0.1,
      }
    );
    if (endOfListRef.current) observer.observe(endOfListRef.current);
    return () => observer.disconnect();
  }, [hasNextPage, isLoading]);
  if (error) {
    return (
      <div className="flex items-center justify-center gap-2 pt-4">
        <CircleAlert className="w-4 h-4" />
        <p className="text-xs">Something went wrong!</p>
      </div>
    );
  }
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 pt-4">
        <LoaderIcon className="w-4 h-4 animate-spin" />
        <p className="text-xs">Fetching rooms...</p>
      </div>
    );
  }
  return (
    <ScrollArea className="h-full relative">
      <div className="grid">
        {data?.pages?.flat().map((room, index) => {
          return (
            <Link className="overflow-hidden" key={index} to={`/${room.id}`}>
              <RoomListItem room={room} />
            </Link>
          );
        })}
        <div className="" ref={endOfListRef}></div>
      </div>
    </ScrollArea>
  );
};
