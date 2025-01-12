import { api } from "@/lib/api";
import { SigninPayload, SigninResponse } from "@/lib/types/auth";
import { useSWRConfig } from "swr";
import useSWRMutation from "swr/mutation";

export const useSignin = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/signin",
    async (url, { arg }: { arg: SigninPayload }) => {
      const res = await api.post(url, arg);
      const data = res.data as SigninResponse;
      return data;
    },
    {
      onSuccess: (data) => {
        mutate("/api/users/me", data, { revalidate: false });
      },
    }
  );
};

export const useSignout = () => {
  const { mutate } = useSWRConfig();
  return useSWRMutation(
    "/api/signout",
    async () => {
      await api.post("/api/signout");
    },
    {
      onError: () => mutate("/api/users/me", null, false),
      onSuccess: () => mutate("/api/users/me", null, false),
    }
  );
};
