import { createContext, useContext, useRef } from "react";

type MessageScrollContext = {
  ref: React.MutableRefObject<HTMLDivElement | null>;
  getScrollContainer: () => Element | null | undefined;
};

const messageScrollContext = createContext<MessageScrollContext>({
  ref: { current: null },
  getScrollContainer: () => null,
});

export const useMessageScroll = () => useContext(messageScrollContext);

export const MessageScrollProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const ref = useRef<HTMLDivElement>(null);
  const getScrollContainer = () =>
    ref.current?.querySelector("[data-radix-scroll-area-viewport]");

  return (
    <messageScrollContext.Provider value={{ ref, getScrollContainer }}>
      {children}
    </messageScrollContext.Provider>
  );
};
