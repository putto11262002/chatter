import { useParams } from "react-router-dom";
import ChatMessageInput from "./message-input";
import MessageArea from "./message-area";
import ChatHeader from "./chat-header";
import { useRoom } from "@/hooks/react-query/chats";
import { LoaderIcon } from "lucide-react";
import { MessageScrollProvider } from "./message-scroll-context";

export default function ChatArea() {
  const params = useParams();
  const roomID = params.roomID!;

  const { data: room } = useRoom(roomID);

  if (!roomID) {
    return (
      <div className="h-full flex justify-center items-center py-4 text-sm">
        <p>No room selected</p>
      </div>
    );
  }

  if (!room) {
    return (
      <div className="h-full flex justify-center items-center py-4 gap-2 text-sm">
        <LoaderIcon className="w-5 h-5 animate-spin" /> <p>Loading room...</p>
      </div>
    );
  }

  return (
    <MessageScrollProvider>
      <div className="flex flex-col h-full">
        <div className="shrink-0">
          <ChatHeader room={room} />
        </div>
        <div className="grow overflow-hidden">
          <MessageArea roomID={roomID} />
        </div>
        <div className="shrink-0 min-h-14 flex-0">
          <ChatMessageInput roomID={roomID} />
        </div>
      </div>
    </MessageScrollProvider>
  );
}
