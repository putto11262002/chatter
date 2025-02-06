import useSWRInfinite from "swr/infinite";
import {
  AddRoomMemberPayload,
  CreateRoomPayload,
  CreateRoomResponse,
  Message,
  Room,
} from "@/lib/types/chat";
import { api } from "@/lib/api";
import useSWR from "swr";
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

export const useCreateRoom = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (arg: CreateRoomPayload) => {
      const res = await api.post("/rooms", arg);
      return res.data as CreateRoomResponse;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "rooms"] });
    },
  });
};

export const useAddRoomMember = ({ roomID }: { roomID: string }) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (arg: AddRoomMemberPayload) => {
      await api.post(`/rooms/${roomID}/members`, arg);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["rooms", roomID] });
    },
  });
};

export const useRemoveRoomMember = ({ roomID }: { roomID: string }) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (arg: { username: string }) => {
      await api.delete(`/rooms/${roomID}/members/${arg.username}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["rooms", roomID] });
    },
  });
};

export const useRoom = (roomID?: string | null) => {
  return useQuery({
    queryKey: ["rooms", roomID],
    queryFn: async () => {
      const res = await api.get(`/rooms/${roomID}`);
      return res.data as Room;
    },
  });
};

export const useInfiniteMyRooms = () => {
  return useInfiniteQuery({
    queryKey: ["users", "me", "rooms"],
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const res = await api.get(
        `/users/me/rooms?offset=${pageParam * 20}&limit=20`
      );
      return res.data as Room[];
    },
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.length < 20) return undefined;
      return lastPageParam + 1;
    },
    staleTime: 1000 * 60 * 5,
  });
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
