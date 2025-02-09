import { createContext, useContext, useState } from "react";

type RoomMemberFormContext = {
  openAddMemberDialog: boolean;
  setOpenAddMemberDialog: (open: boolean) => void;
};

const roomMemberFormContext = createContext<RoomMemberFormContext>({
  openAddMemberDialog: false,
  setOpenAddMemberDialog: () => {},
});

export const useRoomMemberFormContext = () => useContext(roomMemberFormContext);

export function RoomMemberContextProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [openAddMemberDialog, setOpenAddMemberDialog] =
    useState<RoomMemberFormContext["openAddMemberDialog"]>(false);
  return (
    <roomMemberFormContext.Provider
      value={{ openAddMemberDialog, setOpenAddMemberDialog }}
    >
      {children}
    </roomMemberFormContext.Provider>
  );
}
