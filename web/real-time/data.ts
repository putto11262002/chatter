import { MessageType } from "@/lib/types/chat";
import { z } from "zod";

export enum EventName {
  Message = "message",
  ReadMessage = "read_message",
  Online = "online",
  Offline = "offline",
  Typing = "typing",
  IsOnline = "is_online",
}

// Define schemas for each payload type
export const messageBodySchema = z.object({
  id: z.number(),
  room_id: z.string(),
  data: z.string(),
  type: z.nativeEnum(MessageType),
  sender: z.string(),
  sent_at: z.string(),
});

export type MessageBody = z.infer<typeof messageBodySchema>;

export const readMessageBodySchema = z.object({
  room_id: z.string(),
  read_at: z.string(),
  read_by: z.string(),
  last_read_message: z.number(),
});

export type ReadMessageBody = z.infer<typeof readMessageBodySchema>;

export const typingBodySchema = z.object({
  typing: z.boolean(),
  username: z.string(),
  room_id: z.string(),
});

export type TypingBody = z.infer<typeof typingBodySchema>;

export const onlineBodySchema = z.object({
  username: z.string(),
});

export type OnlineBody = z.infer<typeof onlineBodySchema>;

export const offlineBodySchema = onlineBodySchema;

export type OfflineBody = z.infer<typeof offlineBodySchema>;

export const isOnlineBodySchema = onlineBodySchema;

export type IsOnlineBody = z.infer<typeof isOnlineBodySchema>;
