import useSWRMutation from "swr/mutation";
import {
  CreatePrivateChatRequest,
  CreatePrivateChatResponse,
  Message,
  MessageCreateRequest,
} from "@/types/chat";
import { api } from "@/api";
import useSWR, { useSWRConfig } from "swr";
import { UserRoom } from "@/types/user";

export const useCreatePrivateChat = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/chats/private",
    async (url, { arg }: { arg: CreatePrivateChatRequest }) => {
      const res = await api.post(url, arg);
      return res.data as CreatePrivateChatResponse;
    },
    {
      onSuccess: () => mutate("/api/chats/me/rooms"),
    }
  );
};

export const useMyRooms = () => {
  return useSWR(
    "/api/chats/me/rooms",
    async (url) => {
      const res = await api.get(url);
      return res.data as UserRoom[];
    },
    {}
  );
};

export const useSendMessage = (roomID: string) => {
  return useSWRMutation(
    `/api/chats/${roomID}/messages`,
    async (url, { arg }: { arg: MessageCreateRequest }) => {
      const res = await api.post(url, arg);
      return res.data;
    }
  );
};

export const useChatMessageHistory = (roomID: string) => {
  return useSWR(`/api/chats/rooms/${roomID}/messages`, async (url) => {
    const res = await api.get(url);
    return res.data as Message[];
  });
};
