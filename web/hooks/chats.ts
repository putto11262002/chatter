import useSWRMutation from "swr/mutation";
import {
  CreatePrivateChatRequest,
  CreatePrivateChatResponse,
  Message,
  MessageCreateRequest,
  MessageStatus,
} from "@/types/chat";
import { api } from "@/api";
import useSWR, { mutate, useSWRConfig } from "swr";
import { UserRoom } from "@/types/user";
import { useWS } from "./ws-provider";
import { createChatMessagePacket } from "@/lib/ws";
import { useSession } from "@/components/providers/session-provider";

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

export const useSendMessage = (roomID: string) => {
  const ws = useWS();
  const { mutate } = useSWRConfig();
  const session = useSession();
  return useSWRMutation(
    `/api/chats/${roomID}/messages`,
    async (url, { arg }: { arg: Pick<Message, "data" | "type"> }) => {
	const packet = createChatMessagePacket({ ...arg, roomID })
      ws.sendPacket(packet);

      const newMessage: Message = {
	  id: "",
	  correlationID: packet.correlationID,
        data: arg.data,
        type: arg.type,
        sender: session.username,
        sentAt: new Date().toISOString(),
        roomID: roomID,
        status: MessageStatus.PENDING,
      };

      mutate(
        `/api/chats/rooms/${roomID}/messages`,
        async (currentMessages: Message[] | undefined) => [
          newMessage,
          ...(currentMessages ? currentMessages : []),
        ],
        {
          revalidate: false,
        }
      );
    }
  );
};

export const useChatMessageHistory = (roomID: string) => {
  return useSWR(`/api/chats/rooms/${roomID}/messages`, async (url) => {
    const res = await api.get(url);
    return res.data as Message[];
  });
};
