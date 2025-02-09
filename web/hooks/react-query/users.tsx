import { api } from "@/lib/api";
import { CreateUserPayload, UserWithoutSecrets } from "@/types/user";
import { useMutation, useQuery } from "@tanstack/react-query";


export const useRegister = () =>
  useMutation({
    mutationFn: async (data: CreateUserPayload) => {
      return api.post("/users", data);
    }
  })


export const useGetUserByUsername = (username: string) => {
  return useQuery({
    queryKey: ["users", username], queryFn: async ({ signal }) => {
      const res = await api.get(`/users/${username}`, {
        signal,
        validateStatus: (status) =>
          (status >= 200 && status < 300) || status === 404,

      })
      if (res.status !== 200) return null
      return res.data as UserWithoutSecrets
    },
    enabled: username.length > 0
  })
};
