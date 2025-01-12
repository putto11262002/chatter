import { api } from "@/lib/api";
import { CreateUserPayload, UserWithoutSecrets } from "@/lib/types/user";
import { useRef } from "react";
import useSWR from "swr";
import useSWRMutation from "swr/mutation";

export const useMe = () =>
  useSWR(
    "/api/users/me",
    async () => {
      const res = await api.get("/api/users/me");
      const session = res.data;
      return session as UserWithoutSecrets;
    },
    {
      refreshInterval: 1000 * 60 * 5,
    }
  );

export const useSignup = () =>
  useSWRMutation(
    "/api/signup",
    async (url, data: { arg: CreateUserPayload }) => {
      return api.post(url, data.arg);
    },
    {}
  );

export const useGetUserByUsername = (username: string) => {
  const abortControllerRef = useRef<AbortController | null>(null);
  return useSWR(
    username && username !== "" ? `/api/users/${username}` : false,
    async () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      const controller = new AbortController();
      abortControllerRef.current = controller;

      const res = await api.get(`/api/users/${username}`, {
        validateStatus: (status) =>
          (status >= 200 && status < 300) || status === 404,
        signal: controller.signal,
      });
      if (res.status === 404) return null;
      return res.data as UserWithoutSecrets;
    },
    {
      refreshInterval: 1000 * 60 * 5,
    }
  );
};
