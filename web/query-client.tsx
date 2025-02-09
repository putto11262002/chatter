import { QueryClient } from "@tanstack/react-query";
import { APIError } from "./lib/api";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (count, error) => {
        if (error instanceof APIError) {
          if (error.code === 401 || error.code === 403) {
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
