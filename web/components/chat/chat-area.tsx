import { useParams } from "react-router-dom";
import ChatMessageInput from "./message-input";
import MessageArea from "./message-area";
import { useEffect, useState } from "react";
import { useWS } from "@/hooks/ws-provider";

export default function ChatArea() {
  const params = useParams();
  const roomID = params.roomID;
  const [tabActive, setTabActive] = useState(false);

  useEffect(() => {
    window.addEventListener("focus", () => {
      setTabActive(true);
    });
    window.addEventListener("blur", () => {
      setTabActive(false);
    });

    return () => {
      window.removeEventListener("focus", () => {});
      window.removeEventListener("blur", () => {});
      setTabActive(false);
    };
  }, []);

  const { readMessage } = useWS();

  useEffect(() => {
    if (roomID) {
      readMessage(roomID);
    }
  }, [roomID]);

  useEffect(() => {
    if (tabActive && roomID) {
      readMessage(roomID);
    }
  }, [tabActive]);

  if (!roomID) {
    return <div className="h-full w-full bg-muted"></div>;
  }

  return (
    <div className="flex flex-col h-full">
      <div className="grow overflow-hidden">
        <MessageArea roomID={roomID} />
      </div>
      <div className="shrink-0">
        <ChatMessageInput roomID={roomID} />
      </div>
    </div>
  );
}
