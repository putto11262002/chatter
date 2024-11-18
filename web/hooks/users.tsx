import { api } from "@/api";
import {
  Session,
  UserSigninRequest,
  UserSignupRequest,
  UserWithoutSecrets,
} from "@/types/user";
import { useRef } from "react";
import useSWR, { useSWRConfig } from "swr";
import useSWRMutation from "swr/mutation";

export const useSignout = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation("/api/users/signout", async () => {
    mutate("/api/users/me", null, false);
  });
};

export const useMe = () =>
  useSWR(
    "/api/users/me",
    async () => {
      const res = await api.get("/api/users/me");
      const session = res.data;
      return session as Session;
    },
    {}
  );

export const useSignup = () =>
  useSWRMutation(
    "/api/users/signup",
    async (url, data: { arg: UserSignupRequest }) => {
      return api.post(url, data.arg);
    }
  );

export const useSignin = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/users/signin",
    (url, { arg }: { arg: UserSigninRequest }) => {
      return api.post(url, arg);
    },
    { onSuccess: () => mutate("/api/users/me") }
  );
};

export const useGetUserByUsername = (username: string) => {
  const abortControllerRef = useRef<AbortController>(null);
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
    }
  );
};
