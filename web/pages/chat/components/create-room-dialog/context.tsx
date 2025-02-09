import { createContext, useContext, useState } from "react";
import CreateRoomDialog from "./dialog";

type DialogContext = {
  open: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
};

const dialogContext = createContext<DialogContext>({
  open: false,
  setOpen: () => {},
  toggle: () => {},
});

export const useCreateRoomDialog = () => useContext(dialogContext);

export function CreateRoomDialogProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const toggle = () => setOpen((prev) => !prev);

  return (
    <dialogContext.Provider value={{ open, toggle, setOpen }}>
      <CreateRoomDialog />
      {children}
    </dialogContext.Provider>
  );
}
