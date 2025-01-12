import { useParams } from "react-router-dom";
import ChatMessageInput from "./message-input";
import MessageArea from "./message-area";
import ChatHeader from "./chat-header";

export default function ChatArea() {
  const params = useParams();
  const roomID = params.roomID;

  if (!roomID) {
    return <div className="h-full w-full bg-muted"></div>;
  }

  return (
    <div className="flex flex-col h-full">
      <div className="shrink-0">
        <ChatHeader roomID={roomID} />
      </div>
      <div className="grow overflow-hidden">
        <MessageArea roomID={roomID} />
      </div>
      <div className="shrink-0">
        <ChatMessageInput roomID={roomID} />
      </div>
    </div>
  );
}
