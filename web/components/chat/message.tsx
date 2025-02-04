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
        className={cn("px-3 py-2 rounded-lg border text-start", className)}
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
