import { z } from "zod";
import { UserWithoutSecrets } from "./user";

export const signinPayloadSchema = z.object({
  username: z.string().min(1),
  password: z.string().min(1),
});

export type SigninPayload = z.infer<typeof signinPayloadSchema>;

export type SigninResponse = {
  token: string;
  expire_at: string;
  user: UserWithoutSecrets;
};

export type Session = {
  username: string;
  name: string;
};
