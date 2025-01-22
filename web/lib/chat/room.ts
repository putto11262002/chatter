import { Room, RoomMember } from "@/lib/types/chat";

export function getMemberByUsername(
  room: Room,
  username: string
): RoomMember | undefined {
  return room.members.find((member) => member.username === username);
}
