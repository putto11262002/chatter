import { api } from "@/lib/api";
import { SigninPayload, SigninResponse } from "@/lib/types/auth";
import { useSWRConfig } from "swr";
import useSWRMutation from "swr/mutation";

export const useSignin = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/auth/signin",
    async (url, { arg }: { arg: SigninPayload }) => {
      const res = await api.post(url, arg);
      const data = res.data as SigninResponse;
      return data;
    },
    {
      onSuccess: (data) => {
        mutate("/users/me", data, { revalidate: false });
      },
    }
  );
};

export const useSignout = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/auth/signout",
    async () => {
      await api.post("/api/signout");
    },
    {
      onError: () => mutate("/users/me", null, false),
      onSuccess: () => mutate("/users/me", null, false),
    }
  );
};
