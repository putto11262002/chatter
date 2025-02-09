import { Room, RoomMember } from "@/types/chat";

export function getMemberByUsername(
  room: Room,
  username: string
): RoomMember | null {
  return room.members.find((member) => member.username === username) ?? null
}
