import { QueryClient } from "@tanstack/react-query";
import { APIError } from "./lib/api";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (count, error) => {
        if (error instanceof APIError) {
          // Auth errors should not be retried
          if (error.code === 401 || error.code === 403) {
            return false;
          }
          // Resource not found should not be retried
          if (error.code === 404) {
            return false;
          }
        }
        if (count > 3) {
          return false;
        }
        return true;
      },
    },
  },
});
