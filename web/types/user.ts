import { z } from "zod";

export type UserWithoutSecrets = {
  username: string;
  name: string;
};

export const createUserPayloadSchema = z.object({
  username: z.string().min(3),
  password: z.string().min(8),
  name: z.string().min(1),
});

export type CreateUserPayload = z.infer<typeof createUserPayloadSchema>;
