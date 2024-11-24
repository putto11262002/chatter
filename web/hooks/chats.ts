import useSWRMutation from "swr/mutation";
import {
  CreatePrivateChatRequest,
  CreateChatResponse,
  Message,
  Room,
  CreateGroupChatRequest,
  RoomView,
} from "@/types/chat";
import { api } from "@/api";
import useSWR, { useSWRConfig } from "swr";

export const useCreatePrivateChat = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/rooms/private",
    async (url, { arg }: { arg: CreatePrivateChatRequest }) => {
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
    async (url, { arg }: { arg: CreateGroupChatRequest }) => {
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
      return res.data as RoomView[];
    },
    {}
  );
};

export const useReceiveMessage = () => {
  const { mutate } = useSWRConfig();
  return {
    trigger: (message: Message) => {
      mutate(
        `/api/chats/rooms/${message.roomID}/messages`,
        async (currentMessages: Message[] | undefined) => [
          message,
          ...(currentMessages ? currentMessages : []),
        ],
        {
          revalidate: false,
        }
      );
    },
  };
};

export const useChatMessageHistory = (roomID?: string) => {
  return useSWR(
    roomID ? `/api/rooms/${roomID}/messages` : false,
    async (url) => {
      const res = await api.get(url);
      return res.data as Message[];
    }
  );
};
