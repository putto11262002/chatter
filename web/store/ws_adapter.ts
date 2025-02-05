import { Packet } from "@/lib/ws";
import { useRealtimeStore } from "./real-time";
import {
  EventName,
  messageBodySchema,
  readMessageBodySchema,
  typingBodySchema,
} from "@/lib/ws/data";

type EventHandler = (e: Packet) => void;

export function getWSAdapter(
  handlers: Record<string, EventHandler>
): (e: Packet) => void {
  return function (e: Packet): void {
    const handler = handlers[e.type];
    if (!handler) {
      console.error("No handler for packet type", e);
      return;
    }
    handler(e);
  };
}

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
