import useSWRMutation from "swr/mutation";
import {
  CreatePrivateChatPayload,
  CreateChatResponse,
  Message,
  Room,
  CreateGroupChatPayload,
} from "@/lib/types/chat";
import { api } from "@/lib/api";
import useSWR, { useSWRConfig } from "swr";
import { RoomSummary } from "@/lib/types/chat";

export const useCreatePrivateChat = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/rooms/private",
    async (url, { arg }: { arg: CreatePrivateChatPayload }) => {
      const res = await api.post(url, arg);
      return res.data as CreateChatResponse;
    },
    {
      onSuccess: () => mutate("/api/users/me/rooms"),
    }
  );
};

export const useCreateGroupChat = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/rooms/group",
    async (url, { arg }: { arg: CreateGroupChatPayload }) => {
      const res = await api.post(url, arg);
      return res.data as CreateChatResponse;
    },
    {
      onSuccess: () => mutate("/api/users/me/rooms"),
    }
  );
};

export const useRoom = (roomID?: string) => {
  return useSWR(
    roomID ? `/api/rooms/${roomID}` : false,
    async (url) => {
      const res = await api.get(url);
      return res.data as Room;
    },
    {}
  );
};

export const useMyRooms = () => {
  return useSWR(
    "/api/users/me/rooms",
    async (url) => {
      const res = await api.get(url);
      return res.data as RoomSummary[];
    },
    {}
  );
};

export const useChatMessageHistory = (roomID?: string) => {
  return useSWR(
    roomID ? `/api/rooms/${roomID}/messages` : false,
    async (url) => {
      const res = await api.get(url);
      return res.data as Message[];
    },
    {
      refreshInterval: 1000 * 60 * 5, // 5 minutes
    }
  );
};
