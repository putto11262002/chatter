import { ApiError, parseApiError } from "./types/error";
import axios, { CanceledError } from "axios";

export const api = axios.create({
  baseURL: new URL("http://localhost:8080").toString(),
  headers: {
    "Content-Type": "application/json",
  },
  withCredentials: true,
});

export function getError(err: unknown): Error | null {
  if (err instanceof axios.AxiosError) {
    if (err.response) {
      // if it is an ApiError
      return parseApiError(err.response.data);
    }

    if (err.request) {
      console.log("request error", err);
      if (err instanceof CanceledError) {
        return null;
      }
      return new Error("Something went wrong");
    }
  }

  return new Error("Something went wrong");
}

export async function apiRequest<T>(f: () => Promise<T>): Promise<T> {
  try {
    const res = await f();
    return res;
  } catch (err) {
    if (err instanceof Response) {
      if (err.headers.get("Content-Type") == "application/json") {
        const data = await err.json();
        throw parseApiError(data);
      }
      throw new ApiError(500, "internal server error");
    }
    throw new Error("unknown error");
  }
}
