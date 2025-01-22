import { z } from "zod";

export enum MemberRole {
  Owner = "owner",
  Admin = "admin",
  Member = "member",
}
export type RoomMember = {
  username: string;
  role: MemberRole;
  room_id: string;
  last_message_read: number;
};

export type Room = {
  id: string;
  members: RoomMember[];
  name: string;
  last_message_sent_at: string;
  last_message_sent: string;
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

export const createRoomSchema = z.object({
  name: z.string(),
});

export type CreateRoomPayload = z.infer<typeof createRoomSchema>;

export type CreateRoomResponse = {
  id: string;
};

export const addRoomMemberSchema = z.object({
  username: z.string(),
  role: z.nativeEnum(MemberRole),
});

export type AddRoomMemberPayload = z.infer<typeof addRoomMemberSchema>;

export const sendMessagePayloadSchema = z.object({
  data: z.string(),
  type: z.nativeEnum(MessageType),
  sender: z.string(),
  room_id: z.string(),
});

export type SendMessagePayload = z.infer<typeof sendMessagePayloadSchema>;
