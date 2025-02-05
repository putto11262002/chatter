import useSWRMutation from "swr/mutation";
import useSWRInfinite from "swr/infinite";
import {
  AddRoomMemberPayload,
  CreateRoomPayload,
  CreateRoomResponse,
  Message,
  Room,
} from "@/lib/types/chat";
import { api } from "@/lib/api";
import useSWR, { mutate, useSWRConfig } from "swr";
import { useEffect, useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";

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

export const useInfiniteMessages = (roomID: string) => {
  return useInfiniteQuery({
    queryKey: ["rooms", roomID, "messages"],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const res = await api.get(
        `/rooms/${roomID}/messages?offset=${pageParam * 20}&limit=20`
      );
      return res.data as Message[];
    },
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.length < 20) return undefined;
      return lastPageParam + 1;
    },

    staleTime: Infinity,
    select: (data) => ({
      pages: [...data.pages].reverse(),
      pageParams: [...data.pageParams].reverse(),
    }),
  });
};

export const useInfiniteScrollMessageHistory = (roomID?: string) => {
  const limit = 20;

  const returned = useSWRInfinite(
    (index, prev) => {
      if (!roomID) return false;
      if (prev && prev.length < 1) return null;
      return `/rooms/${roomID}/messages?offset=${index * limit}&limit=${limit}`;
    },
    async (url) => {
      const res = await api.get(url);
      return res.data as Message[];
    },
    {
      revalidateAll: false,
      revalidateFirstPage: false,
    }
  );
  return {
    pages: returned,
    ...returned,
  };
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
