import { queryClient } from "@/query-client";
import { QueryClientProvider } from "@tanstack/react-query";
import { Outlet } from "react-router-dom";

export default function RootLayout() {
  return (
    <QueryClientProvider client={queryClient}>
      <Outlet />
    </QueryClientProvider>
  );
}
