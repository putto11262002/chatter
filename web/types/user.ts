import { z } from "zod";
export type UserWithoutSecrets = {
  username: string;
  name: string;
};

export const UserSignupRequestSchema = z.object({
  username: z.string().min(3),
  password: z.string().min(8),
  name: z.string().min(1),
});

export type UserSignupRequest = z.infer<typeof UserSignupRequestSchema>;

export const UserSigninRequestSchema = z.object({
  username: z.string().min(1),
  password: z.string().min(1),
});

export type UserSigninRequest = z.infer<typeof UserSigninRequestSchema>;

export type Session = {
  username: string;
  name: string;
};

export type UserRoom = {
  roomID: string;
  roomName: string;
  username: string;
};
