import { z } from "zod";

export const eventSchema = z.object({
  type: z.string(),
  payload: z.unknown(),
  id: z.number().optional(),
});

export type WSEvent = z.infer<typeof eventSchema>;

export function encodeEvent(packet: WSEvent): string {
  return JSON.stringify(packet);
}

export function decodeEvent(raw: string): WSEvent | null {
  try {
    const parsed = JSON.parse(raw);
    const packet = eventSchema.parse(parsed);
    return packet;
  } catch {
    return null;
  }
}

export function craeteEvent(type: string, body: unknown): WSEvent {
  return {
    type,
    payload: body,
    id: Math.floor(Math.random() * 65536),
  };
}
