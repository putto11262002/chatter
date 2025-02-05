import { useRealtimeStore } from "@/store/real-time";
import {
  EventName,
  messageBodySchema,
  readMessageBodySchema,
  typingBodySchema,
} from "./data";
import { EventHandler } from "./context";

const onMessage: EventHandler = (packet) => {
  const message = messageBodySchema.parse(packet.payload);

  useRealtimeStore.getState().addMessage(message.room_id, message);
  // if the message belongs to the user
};

const onTyping: EventHandler = (e) => {
  const { username, typing, room_id } = typingBodySchema.parse(e.payload);
  useRealtimeStore.getState().setUserTyping(username, typing ? room_id : null);
};

const onMessageRead: EventHandler = (e) => {
  const readMessage = readMessageBodySchema.parse(e.payload);
  console.log("recieved read message event", readMessage);
};

export const storeEventHandlers: Record<string, EventHandler> = {
  [EventName.Message]: onMessage,
  [EventName.Typing]: onTyping,
  [EventName.ReadMessage]: onMessageRead,
};
