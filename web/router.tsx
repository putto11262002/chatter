import { createBrowserRouter, Outlet } from "react-router-dom";
import Signup from "./pages/signup";
import Signin from "./pages/signin";
import SessionProvider from "./components/providers/session-provider";
import { SWRConfig } from "swr";
import { apiErrorMiddleware, sessionMiddleware } from "./hooks/swr-middlewares";
import ChatLayout from "./components/layouts/chat-layout";
import ChatArea from "./components/chat/chat-area";

export const router = createBrowserRouter([
  {
    element: (
      <SWRConfig value={{ use: [apiErrorMiddleware] }}>
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
              <SWRConfig value={{ use: [sessionMiddleware] }}>
                <Outlet />
              </SWRConfig>
            ),
            children: [
              {
                path: "/",
                element: <ChatLayout />,
                children: [
                  {
                    path: "/:roomID",
                    element: <ChatArea />,
                  },
                ],
              },
            ],
          },
        ],
      },
    ],
  },
]);
