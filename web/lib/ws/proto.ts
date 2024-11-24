import { MessageSchema, MessageType } from "@/types/chat";
import { base64Decode, base64Encode, generateCorrelationID } from "./utils";
import { z } from "zod";

export enum PacketType {
  SendMessageRequestPacket = 1,
  SendMessageResponsePacket = 2,
  BroadcastMessagePacket = 3,
  ReadMessagePacket = 4,
  BroadcastReadMessagePacket = 5,
  PresencePacket = 6,
  TypingEventPacket = 7,
}

export const PacketSchema = z.object({
  correlationID: z.number(),
  type: z.number(),
  payload: z.string(),
  from: z.string(),
  sentAt: z.string().optional(),
});

export type Packet = z.infer<typeof PacketSchema>;

export function encodePacket(type: PacketType, data: string): Packet {
  return {
    correlationID: generateCorrelationID(),
    type,
    payload: base64Encode(data),
    from: "",
    sentAt: new Date().toISOString(),
  };
}

export function decodePacket(raw: string): Packet {
  const obj = JSON.parse(raw);
  const validation = PacketSchema.safeParse(obj);
  if (!validation.success) {
    throw new Error(
      "invalid packet format: " + JSON.stringify(validation.error.format())
    );
  }

  const packet = validation.data;
  packet.payload = JSON.parse(base64Decode(packet.payload));
  return packet;
}

// Define schemas for each payload type
export const SendMessageRequestPayloadSchema = z.object({
  roomID: z.string(),
  type: z.nativeEnum(MessageType),
  data: z.string(),
});
export type SendMessageRequestPayload = z.infer<
  typeof SendMessageRequestPayloadSchema
>;

export const SendMessageResponsePayloadSchema = z.object({
  code: z.number(),
  roomID: z.string(),
  messageID: z.number(),
  sentAt: z.string(),
});
export type SendMessageResponsePayload = z.infer<
  typeof SendMessageResponsePayloadSchema
>;

export const ReadMessagePayloadSchema = z.object({
  roomID: z.string(),
});

export type ReadMessagePayload = z.infer<typeof ReadMessagePayloadSchema>;

export const BroadcastReadMessagePayloadSchema = z.object({
  roomID: z.string(),
  username: z.string(),
  messageID: z.number(),
  readAt: z.string(),
});
export type BroadcastReadMessagePayload = z.infer<
  typeof BroadcastReadMessagePayloadSchema
>;

export const BroadcastMessagePayloadSchema = MessageSchema;

export type BroadcastMessagePayload = z.infer<
  typeof BroadcastMessagePayloadSchema
>;

export const TypingEventPayloadSchema = z.object({
  roomID: z.string(),
  typing: z.boolean(),
  username: z.string(),
});
export type TypingEventPayload = z.infer<typeof TypingEventPayloadSchema>;

export const PresencePayloadSchema = z.object({
  username: z.string(),
  presence: z.boolean(),
});
export type PresencePayload = z.infer<typeof PresencePayloadSchema>;
