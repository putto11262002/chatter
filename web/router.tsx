import { createBrowserRouter, Outlet } from "react-router-dom";
import Signup from "./pages/signup";
import Signin from "./pages/signin";
import SessionProvider from "./components/providers/session-provider";
import { SWRConfig } from "swr";
import { ChatProvider } from "@/components/context/chat/provider";
import { clearSesssionOnAuthError } from "./lib/swr/middlewares";
import ChatPage from "./pages/chat";

export const router = createBrowserRouter([
  {
    element: (
      <SWRConfig
        value={{
          revalidateOnMount: true,
          revalidateIfStale: false,
          refreshInterval: 1000 * 60 * 5,
        }}
      >
        <Outlet />
      </SWRConfig>
    ),
    children: [
      {
        path: "/signup",
        element: <Signup />,
      },
      {
        path: "/signin",
        element: <Signin />,
      },
      {
        element: <SessionProvider />,
        children: [
          {
            element: (
              <SWRConfig value={{ use: [clearSesssionOnAuthError] }}>
                <ChatProvider>
                  <Outlet />
                </ChatProvider>
              </SWRConfig>
            ),

            children: [
              {
                path: "/:roomID?",
                element: <ChatPage />,
              },
            ],
          },
        ],
      },
    ],
  },
]);
