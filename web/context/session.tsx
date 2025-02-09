import { useGetCurrentUser } from "@/hooks/react-query/auth";
import { queryClient } from "@/query-client";
import { Session } from "@/types/auth";
import React, { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const sessionContext = React.createContext<Session>({ username: "", name: "" });

export const useSession = () => React.useContext(sessionContext);

// Used for non-react code
export const getSession = () =>
  queryClient.getQueryData<Session | null>(["users", "me"]);

export const SessionProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const { data, isLoading } = useGetCurrentUser();

  const navigate = useNavigate();

  useEffect(() => {
    if (!data && !isLoading) {
      navigate("/auth/signin");
    }
  }, [data, isLoading, navigate]);

  if (!data || isLoading) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-background">
        Chatter...
      </div>
    );
  }

  return (
    <sessionContext.Provider value={data}>{children}</sessionContext.Provider>
  );
};
