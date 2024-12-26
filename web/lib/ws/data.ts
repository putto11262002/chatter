import { MessageType } from "@/types/chat";
import { z } from "zod";

export enum PacketType {
  Message = "message",
  ReadMessage = "read_message",
  Online = "online",
  Offline = "offline",
  Typing = "typing",
}

// Define schemas for each payload type
export const MessageBody = z.object({
  id: z.number(),
  room_id: z.string(),
  data: z.string(),
  type: z.nativeEnum(MessageType),
  sender: z.string(),
  sent_at: z.string(),
});

export const ReadMessageBody = z.object({
  room_id: z.string(),
  message_id: z.number(),
  read_at: z.string(),
  last_read_message: z.number(),
});

export const TypingBody = z.object({
  typing: z.boolean(),
  username: z.string(),
  room_id: z.string(),
});
