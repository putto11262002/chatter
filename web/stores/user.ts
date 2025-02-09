import { StateCreator } from "zustand";

export type UserRealtimeInfo = {
  username: string;
  // typing is set to the room id in which there are typing in
  // or null if the user is not typing
  typing: string | null;
  online: boolean;
};

export type UserSlice = {
  users: Record<string, UserRealtimeInfo>;
  setUserOnline: (username: string, online: boolean) => void;
  setUserTyping: (username: string, typing: string | null) => void;
  setUser: (username: string, user: UserRealtimeInfo) => void;
};

export const createUserSlice: StateCreator<
  UserSlice,
  [["zustand/devtools", never]]
> = (set) => ({
  users: {},
  setUserOnline: (username, online) => {
    set((state) => ({
      users: {
        ...state.users,
        [username]: { ...state.users[username], username, online },
      },
    }));
  },
  setUserTyping: (username, typing) => {
    set((state) => ({
      users: {
        ...state.users,
        [username]: {
          typing,
          online: true,
          username,
        },
      },
    }));
  },
  setUser: (username, user) => {
    set((state) => ({
      users: {
        ...state.users,
        [username]: user,
      },
    }));
  },
});
