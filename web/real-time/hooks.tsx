import { useSession } from "@/components/providers/session-provider";
import { ReadyState } from "./ws";
import { useWS } from "./context";
import {
  EventName,
  IsOnlineBody,
  MessageBody,
  TypingBody,
  typingBodySchema,
} from "./data";
import { useRealtimeStore } from "@/store/real-time";
import { UserRealtimeInfo } from "@/store/user";

export const useSendMessage = () => {
  const { ws } = useWS();
  return {
    available: ws.readyState === ReadyState.Open,
    send: function sendMessage(
      message: Pick<MessageBody, "data" | "room_id" | "type">
    ) {
      ws.sendPacket({
        type: EventName.Message,
        payload: message,
      });
    },
  };
};

export const useTyping = () => {
  const { ws, readyState } = useWS();
  const { username } = useSession();
  function emitTyping(roomID: string, typing: boolean) {
    const typingEvent: TypingBody = {
      room_id: roomID,
      username: username,
      typing: typing,
    };
    ws.sendPacket({
      type: EventName.Typing,
      payload: typingBodySchema.parse(typingEvent),
    });
    useRealtimeStore.getState().setUserTyping(username, typing ? roomID : null);
  }

  return {
    available: readyState === ReadyState.Open,
    startTyping: (roomID: string) => emitTyping(roomID, true),
    stopTyping: (roomID: string) => emitTyping(roomID, false),
  };
};

export const useRealtimeUserInfo = () => {
  const users = useRealtimeStore((state) => state.users);
  const { ws } = useWS();
  const session = useSession();
  return {
    get: (username: string): UserRealtimeInfo => {
      if (username === session.username) {
        return {
          username: session.username,
          online: true,
          typing: users[username]?.typing || null,
        };
      }
      const realtimeInfo = users[username];
      if (!realtimeInfo) {
        const payload: IsOnlineBody = {
          username,
        };
        ws.sendPacket({ type: EventName.IsOnline, payload });
        return { username, typing: null, online: false };
      }
      return realtimeInfo;
    },
  };
};
