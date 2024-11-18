import { getError } from "@/api";
import { ApiError } from "@/types/error";
import { Middleware, SWRHook, useSWRConfig } from "swr";

export const apiErrorMiddleware: Middleware =
  (useSWRNext: SWRHook) => (key, fetcher, config) => {
    const extendedFetcher = async (...args: unknown[]) => {
      try {
        const res = await fetcher!(...args);
        return res;
      } catch (err) {
        throw getError(err);
      }
    };
    return useSWRNext(key, fetcher ? extendedFetcher : null, config);
  };

export const sessionMiddleware: Middleware =
  (useSWRNext: SWRHook) => (key, fetcher, config) => {
    const { mutate } = useSWRConfig();
    const extendedFetcher = async (...args: unknown[]) => {
      try {
        const res = await fetcher!(...args);
        return res;
      } catch (err) {
        if (err instanceof ApiError && err.code === 401) {
          mutate("/api/users/me", undefined);
        }
        throw err;
      }
    };
    return useSWRNext(key, fetcher ? extendedFetcher : null, config);
  };
