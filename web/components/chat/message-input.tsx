import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { MessageType } from "@/types/chat";
import { Send } from "lucide-react";
import { useForm } from "react-hook-form";
import { Form } from "../ui/form";
import { useWS } from "@/hooks/ws-provider";
import { useEffect, useRef, useState } from "react";

export default function ChatMessageInput({ roomID }: { roomID: string }) {
  const { sendMessage, emitTypingEvent } = useWS();
  const timeRef = useRef<Timer>(null);
  const form = useForm();
  const message = form.watch("message");
  const [typing, setTyping] = useState(false);

  const handleInput = () => {
    if (timeRef.current) {
      clearTimeout(timeRef.current);
    }
    setTyping(true);
    if (!typing) {
      emitTypingEvent(roomID, true);
    }
    timeRef.current = setTimeout(() => {
      setTyping(false);
      emitTypingEvent(roomID, false);
    }, 1000);
  };

  const onSumit = (data: string) => {
    sendMessage({ data, roomID, type: MessageType.TEXT });
    form.reset();
  };

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(
          ({ message }) => message && onSumit(message)
        )}
        className="flex py-2 px-2 gap-2 border-t"
      >
        <Input
          onInput={handleInput}
          className="grow"
          {...form.register("message")}
        />
        <Button disabled={!message} className="" size="icon" type="submit">
          <Send className="w-6 h-6" />
        </Button>
      </form>
    </Form>
  );
}
