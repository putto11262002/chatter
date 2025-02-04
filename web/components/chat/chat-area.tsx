import { useParams } from "react-router-dom";
import ChatMessageInput from "./message-input";
import MessageArea from "./message-area";
import ChatHeader from "./chat-header";
import { useRoom } from "@/hooks/chats";
import { Loader2 } from "lucide-react";
import { StrictMode } from "react";

export default function ChatArea() {
  const params = useParams();
  const roomID = params.roomID!;

  const { data: room } = useRoom(roomID);

  if (!room) {
    return (
      <div className="h-full flex justify-center items-center py-4">
        <Loader2 className="w-4 h-4 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="shrink-0">
        <ChatHeader room={room} />
      </div>
      <div className="grow overflow-hidden">
        <MessageArea roomID={roomID} />
      </div>
      <div className="shrink-0 min-h-14 flex-0">
        <StrictMode>
          <ChatMessageInput roomID={roomID} />
        </StrictMode>
      </div>
    </div>
  );
}
