import { useRealtimeStore } from "@/stores/real-time";
import {
  EventName,
  messageBodySchema,
  offlineBodySchema,
  onlineBodySchema,
  readMessageBodySchema,
  typingBodySchema,
} from "@/types/ws";
import { queryClient } from "@/query-client";
import { EventHandler } from "@/lib/ws";

const onMessage: EventHandler = (packet) => {
  const message = messageBodySchema.parse(packet.payload);

  useRealtimeStore.getState().addMessage(message.room_id, message);
  // if the message belongs to the user set the new read pointer for the user

  // update the room last message sent
  queryClient.invalidateQueries({ queryKey: ["room", message.room_id] });
  queryClient.invalidateQueries({ queryKey: ["users", "me", "rooms"] });
};

const onTyping: EventHandler = (e) => {
  const { username, typing, room_id } = typingBodySchema.parse(e.payload);
  useRealtimeStore.getState().setUserTyping(username, typing ? room_id : null);
};

const onMessageRead: EventHandler = (e) => {
  const readMessage = readMessageBodySchema.parse(e.payload);
  console.log("recieved read message event", readMessage);
};

const onOnline: EventHandler = (e) => {
  const { username } = onlineBodySchema.parse(e.payload);
  useRealtimeStore.getState().setUserOnline(username, true);
};

const onOffline: EventHandler = (e) => {
  const { username } = offlineBodySchema.parse(e.payload);
  useRealtimeStore.getState().setUserOnline(username, false);
};

export const eventHandlers: Record<string, EventHandler> = {
  [EventName.Message]: onMessage,
  [EventName.Typing]: onTyping,
  [EventName.ReadMessage]: onMessageRead,
  [EventName.Online]: onOnline,
  [EventName.Offline]: onOffline,
};
