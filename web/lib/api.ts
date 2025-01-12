import axios from "axios";
import { z } from "zod";

export class APIError extends Error {
  constructor(public code: number, public err: string) {
    super(err);
  }
}

export const apiErorSchema = z.object({
  code: z.number(),
  error: z.string(),
});

export const api = axios.create({
  baseURL: new URL("http://localhost:8081").toString(),
  headers: {
    "Content-Type": "application/json",
  },
  withCredentials: true,
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error("API error", error);
    if (axios.isAxiosError(error)) {
      if (error.response) {
        const val = apiErorSchema.safeParse(error.response.data);
        if (val.success) {
          throw new APIError(val.data.code, val.data.error);
        } else {
          throw new Error("Unknown error");
        }
      }
      if (error.request) {
        throw new Error("Failed to send request");
      }

      if (axios.isCancel(error)) {
        throw new Error("Request was canceled");
      }

      throw new Error("Unknown error");
    } else {
      throw new Error("Unknown error");
    }
  }
);
