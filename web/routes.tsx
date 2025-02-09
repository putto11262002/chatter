import { createBrowserRouter } from "react-router-dom";
import RootLayout from "./pages/layout";
import AuthLayout from "./pages/auth/layout";
import Signup from "./pages/auth/signup";
import Signin from "./pages/auth/signin";
import ProtectedLayut from "./pages/protected/layout";
import ChatLayout from "./pages/chat/layout";
import ChatPage from "./pages/chat";
import RoomSettingsPage from "./pages/room-settings";

export const router = createBrowserRouter([
  {
    element: <RootLayout />,

    children: [
      {
        path: "/auth",
        element: <AuthLayout />,
        children: [
          {
            path: "signup",
            element: <Signup />,
          },
          {
            path: "signin",
            element: <Signin />,
          },
        ],
      },
      {
        element: <ProtectedLayut />,
        children: [
          {
            element: <ChatLayout />,
            children: [{ path: "/:roomID?", element: <ChatPage /> }],
          },

          {
            path: "/rooms/:roomID/settings",
            element: <RoomSettingsPage />,
          },
        ],
      },
    ],
  },
]);
