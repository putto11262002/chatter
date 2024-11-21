import { z } from "zod";

export type Room = {
  id: string;
  users: RoomUser[];
  type: string;
};

export type RoomUser = {
  username: string;
  roomID: string;
  roomName: string;
};

export type CreatePrivateChatRequest = {
  other: string;
};

export type CreatePrivateChatResponse = {
  id: string;
};

export enum MessageType {
  TEXT = 1,
}

export type MessageCreateRequest = {
  data: string;
  type: MessageType;
  roomID: string;
};

export enum MessageStatus {
  PENDING = 0,
  SENT = 1,
  READ = 2,
  FAIL = 3,
}

export const MessageStatusLabels: Record<MessageStatus, string> = {
  [MessageStatus.PENDING]: "Pending",
  [MessageStatus.SENT]: "Sent",
  [MessageStatus.READ]: "Read",
  [MessageStatus.FAIL]: "Fail",
};

export type Message = {
  id: string;
  data: string;
  sender: string;
  type: MessageType;
  sentAt: string;
  roomID: string;
  status: MessageStatus;
  correlationID?: number;
};

export const MessageSchema = z.object({
  id: z.string(),
  data: z.string(),
  sender: z.string(),
  type: z.nativeEnum(MessageType),
  sentAt: z.string(),
  roomID: z.string(),
  status: z.nativeEnum(MessageStatus),
});

export type MessageStatusUpdate = {
  messageID: string;
  status: MessageStatus;
  roomID: string;
};

export const MessageStatusUpdateSchema = z.object({
  roomID: z.string(),
  messageID: z.string(),
  status: z.nativeEnum(MessageStatus),
});

export type ReadRoomMessagesPacketPayload = {
  roomID: string;
  readBy: string;
};

export const ReadRoomMessagePacketPayloadSchema = z.object({
  roomID: z.string(),
  readBy: z.string(),
});

export const TypingEventPacketPayloadSchema = z.object({
  roomID: z.string(),
  user: z.string(),
  typing: z.boolean(),
});

export type TypingEventPacketPayload = z.infer<
  typeof TypingEventPacketPayloadSchema
>;
