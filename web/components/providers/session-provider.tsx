import { useMe } from "@/hooks/users";
import { Session } from "@/lib/types/auth";
import React, { useEffect } from "react";
import { Outlet, useNavigate } from "react-router-dom";

const sessionContext = React.createContext<Session>({ username: "", name: "" });

export const useSession = () => React.useContext(sessionContext);

const SessionProvider = () => {
  const { data, isLoading, error } = useMe();

  const navigate = useNavigate();

  useEffect(() => {
    if (!data && !isLoading) {
      navigate("/signin");
    }
    // if (data && !isLoading) {
    //   navigate("/");
    // }
  }, [data, isLoading, error]);

  if (!data || isLoading) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-background">
        Chatter...
      </div>
    );
  }

  return (
    <sessionContext.Provider value={data}>{<Outlet />}</sessionContext.Provider>
  );
};

export default SessionProvider;
