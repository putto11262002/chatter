import { z } from "zod";

export type RoomView = {
  roomID: string;
  roomName: string;
  users: string[];
};

export type Room = {
  ID: string;
  Type: string;
  users: RoomUser[];
};

export type RoomUser = {
  username: string;
  roomID: string;
  roomName: string;
  lastMessageRead: number;
};

export type CreatePrivateChatRequest = {
  other: string;
};

export type CreateGroupChatRequest = {
  name: string;
  users: string[];
};

export type CreateChatResponse = {
  id: string;
};

export enum MessageType {
  TEXT = 1,
}

export enum MessageStatus {
  PENDING = 0,
  SENT = 1,
  FAIL = 2,
}

export const MessageInteractionSchema = z.object({
  messageID: z.number(),
  username: z.string(),
  readAt: z.string(),
});

export type MessageInteraction = z.infer<typeof MessageInteractionSchema>;

export const MessageSchema = z.object({
  id: z.number(),
  data: z.string(),
  sender: z.string(),
  type: z.nativeEnum(MessageType),
  sentAt: z.string(),
  roomID: z.string(),
  interactions: z.array(MessageInteractionSchema),
  correlationID: z.number().optional(),
});

export type Message = z.infer<typeof MessageSchema>;
