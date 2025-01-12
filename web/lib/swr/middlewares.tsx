import { Middleware, SWRHook, useSWRConfig } from "swr";
import { APIError } from "../api";

// If the request returns an unauthenticated error, the middleware will clear the session
export const clearSesssionOnAuthError: Middleware =
  (useSWRNext: SWRHook) => (key, fetcher, config) => {
    const { mutate } = useSWRConfig();
    const extendedFetcher = async (...args: unknown[]) => {
      try {
        const res = await fetcher!(...args);
        return res;
      } catch (err) {
        if (err instanceof APIError && err.code === 401) {
          mutate("/api/users/me", undefined);
        }
        throw err;
      }
    };
    return useSWRNext(key, fetcher ? extendedFetcher : null, config);
  };
