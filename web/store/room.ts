import { StateCreator } from "zustand";

export type RoomRealtimeInfo = {
  roomID: string;
  lastMessageSent?: number;
  lastMessageSentAt?: string;
  lastMessageSentData?: string;
};

export type RoomSlice = {
  rooms: Record<string, RoomRealtimeInfo>;
  setRoom: (room: RoomRealtimeInfo) => void;
};

export const createRoomSlice: StateCreator<RoomSlice> = (set) => ({
  rooms: {},
  setRoom: (room) => {
    set((state) => ({ rooms: { ...state.rooms, [room.roomID]: room } }));
  },
});
