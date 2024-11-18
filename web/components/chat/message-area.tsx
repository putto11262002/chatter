import { useChatMessageHistory } from "@/hooks/chats";
import { useSession } from "../providers/session-provider";
import Alert from "../alert";
import { Loader2 } from "lucide-react";
import Message from "./message";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";
import { ScrollArea } from "../ui/scroll-area";

export default function MessageArea({ roomID }: { roomID: string }) {
  const session = useSession();
  const { data, isLoading, error } = useChatMessageHistory(roomID);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const messages = data?.slice().reverse();

  useEffect(() => {
    if (!scrollAreaRef.current) return;
    
    // Allow the DOM to update before scrolling
    requestAnimationFrame(() => {
      const scrollContainer = scrollAreaRef.current?.querySelector('[data-radix-scroll-area-viewport]');
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
          return (
            <div
              className={cn(
                "flex",
                myMessage ? "justify-end" : "justify-start"
              )}
              key={index}
            >
              <div className="max-w-[70%]">
                <Message
                  message={message}
                  className={myMessage ? "bg-gray-200" : ""}
                />
              </div>
            </div>
          );
        })}
      </div>
    </ScrollArea>
  );
}
