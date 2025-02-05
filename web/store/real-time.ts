import { create } from "zustand";
import { createUserSlice, UserSlice } from "./user";
import { createMessageSlice, MessageSlice } from "./message";
import { devtools } from "zustand/middleware";

export type RealtimeStore = UserSlice & MessageSlice;

export const useRealtimeStore = create<RealtimeStore>()(
  devtools((...a) => ({
    ...createUserSlice(...a),
    ...createMessageSlice(...a),
  }))
);
