import { z } from "zod";

export enum ChatType {
  Private = 0,
  Group = 1,
}

export type RoomMember = {
  username: string;
  room_id: string;
  room_name: string;
  last_message_read: number;
};

export type Room = {
  id: string;
  members: RoomMember[];
  type: ChatType;
  last_message_sent_at: string;
  last_message_sent: string;
};

export type RoomSummary = {
  id: string;
  name: string;
  members: string[];
  last_message_read: number;
};

export enum MessageType {
  Text = 1,
}

export type Message = {
  id: number;
  type: MessageType;
  data: string;
  room_id: string;
  sender: string;
  sent_at: string;
};

export const createPrivateChatPayloadSchema = z.object({
  other: z.string(),
});

export type CreatePrivateChatPayload = z.infer<
  typeof createPrivateChatPayloadSchema
>;

export const createGroupChatPayloadSchema = z.object({
  members: z.array(z.string()),
  name: z.string(),
});

export type CreateGroupChatPayload = z.infer<
  typeof createGroupChatPayloadSchema
>;

export type CreateChatResponse = {
  id: string;
};

export const sendMessagePayloadSchema = z.object({
  data: z.string(),
  type: z.nativeEnum(MessageType),
  sender: z.string(),
  room_id: z.string(),
});

export type SendMessagePayload = z.infer<typeof sendMessagePayloadSchema>;
