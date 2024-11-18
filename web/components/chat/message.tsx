import { cn } from "@/lib/utils";
import { Message as _Message, MessageType } from "@/types/chat";
export default function Message({
  message,
  className,
}: {
  message: _Message;
  className?: string;
}) {
  if (message.type === MessageType.TEXT) {
    return (
      <div className={cn("px-3 py-2 rounded-lg border", className)}>
        {message.data}
      </div>
    );
  }

  return <div className="">Unsupported message type</div>;
}
