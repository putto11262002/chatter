import { createBrowserRouter, Outlet } from "react-router-dom";
import Signup from "./pages/signup";
import Signin from "./pages/signin";
import SessionProvider from "./components/providers/session-provider";
import { SWRConfig } from "swr";
import { apiErrorMiddleware, sessionMiddleware } from "./hooks/swr-middlewares";
import { ChatProvider } from "./hooks/chat/provider";

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
                element: <ChatProvider />,
                children: [
                  {
                    path: "/",
                    element: <div>Hi there</div>,
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
