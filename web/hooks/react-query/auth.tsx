import { api } from "@/lib/api";
import { SigninPayload, SigninResponse } from "@/types/auth";
import { UserWithoutSecrets } from "@/types/user";
import { useMutation, useQuery, useQueryClient, } from "@tanstack/react-query";

export const useSignin = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ payload }: { payload: SigninPayload }) => {
      const res = await api.post("/auth/signin", payload)
      const data = res.data as SigninResponse
      return data
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["users", "me"], data.user)
    }
  })
};

export const useGetCurrentUser = () => {
  return useQuery({
    queryKey: ["users", "me"], queryFn: async () => {
      const res = await api.get("/users/me");
      const user = res.data;
      return user as UserWithoutSecrets
    }
  })
}

export const useSignout = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      await api.post("/auth/signout")
    }, onSettled: () => {
      queryClient.setQueryData(["users", "me"], null)
    }
  })

};
