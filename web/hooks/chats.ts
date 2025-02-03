import useSWRMutation from "swr/mutation";
import {
  AddRoomMemberPayload,
  CreateRoomPayload,
  CreateRoomResponse,
  Message,
  Room,
} from "@/lib/types/chat";
import { api } from "@/lib/api";
import useSWR, { useSWRConfig } from "swr";

export const useCreateRoom = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/rooms",
    async (url: string, { arg }: { arg: CreateRoomPayload }) => {
      const res = await api.post(url, arg);
      return res.data as CreateRoomResponse;
    },
    {
      onSuccess: () => mutate("/users/me/rooms"),
    }
  );
};

export const useAddRoomMember = ({ roomID }: { roomID: string }) => {
  return useSWRMutation(
    `/rooms/${roomID}`,
    async (_: string, { arg }: { arg: AddRoomMemberPayload }) => {
      await api.post(`/rooms/${roomID}/members`, arg);
    }
  );
};

export const useRemoveRoomMember = ({ roomID }: { roomID: string }) => {
  return useSWRMutation(
    `/rooms/${roomID}`,
    async (_: string, { arg }: { arg: { username: string } }) => {
      await api.delete(`/rooms/${roomID}/members/${arg.username}`);
    }
  );
};

export const useRoom = (roomID?: string | null) => {
  return useSWR(
    roomID ? `/rooms/${roomID}` : false,
    async (url) => {
      const res = await api.get(url);
      return res.data as Room;
    },
    {}
  );
};

export const useMyRooms = () => {
  return useSWR(
    "/users/me/rooms",
    async (url) => {
      const res = await api.get(url);
      return res.data as Room[];
    },
    {}
  );
};

export const useChatMessageHistory = (roomID?: string) => {
  return useSWR(
    roomID ? `/rooms/${roomID}/messages` : false,
    async (url) => {
      const res = await api.get(url);
      return res.data as Message[];
    },
    {
      refreshInterval: 0,
      revalidateOnFocus: false,
    }
  );
};
