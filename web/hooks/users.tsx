import { api } from "@/lib/api";
import { CreateUserPayload, UserWithoutSecrets } from "@/lib/types/user";
import { useRef } from "react";
import useSWR from "swr";
import useSWRMutation from "swr/mutation";

export const useMe = () =>
  useSWR("/users/me", async () => {
    const res = await api.get("/users/me");
    const session = res.data;
    return session as UserWithoutSecrets;
  });

export const useRegister = () =>
  useSWRMutation(
    "/users",
    async (url, data: { arg: CreateUserPayload }) => {
      return api.post(url, data.arg);
    },
    {}
  );

export const useGetUserByUsername = (username: string) => {
  const abortControllerRef = useRef<AbortController | null>(null);
  return useSWR(
    username && username !== "" ? `/users/${username}` : false,
    async () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      const controller = new AbortController();
      abortControllerRef.current = controller;

      const res = await api.get(`/users/${username}`, {
        validateStatus: (status) =>
          (status >= 200 && status < 300) || status === 404,
        signal: controller.signal,
      });
      if (res.status === 404) return null;
      return res.data as UserWithoutSecrets;
    }
  );
};
