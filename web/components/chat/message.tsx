import { cn } from "@/lib/utils";
import { Message as _Message, MessageType } from "@/lib/types/chat";
import { forwardRef } from "react";

const Message = forwardRef<
  HTMLDivElement,
  { message: _Message; className?: string }
>(({ message, className }, ref) => {
  if (message.type === MessageType.Text) {
    return (
      <div
        ref={ref}
        className={cn(
          "w-full min-w-0 overflow-hidden px-3 py-2 rounded-lg border text-start break-words whitespace-pre-wrap",

          className
        )}
      >
        {message.data}
      </div>
    );
  }

  return (
    <div ref={ref} className={className}>
      Unsupported message type
    </div>
  );
});

Message.displayName = "Message";

export default Message;
