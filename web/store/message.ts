import { Message } from "@/lib/types/chat";
import { StateCreator } from "zustand";

export type MessageSlice = {
  messages: Record<string, Message[]>;
  addMessage: (roomId: string, message: Message) => void;
  clearMessages: (roomId: string) => void;
  clearAllMessages: () => void;
};

export const createMessageSlice: StateCreator<
  MessageSlice,
  [["zustand/devtools", never]]
> = (set, get) => ({
  messages: {},
  addMessage: (roomId, message) => {
    set((state) => ({
      messages: {
        ...state.messages,
        [roomId]: [...(state.messages[roomId] || []), message],
      },
    }));
  },
  clearMessages: (roomId) => {
    set((state) => ({
      messages: {
        ...state.messages,
        [roomId]: [],
      },
    }));
  },
  clearAllMessages: () => {
    set({ messages: {} });
  },
});
