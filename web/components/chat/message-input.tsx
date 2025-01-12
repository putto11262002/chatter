import { Button } from "@/components/ui/button";
import { Send } from "lucide-react";
import { useForm } from "react-hook-form";
import { Form } from "../ui/form";
import { useRef, useState } from "react";
import { Textarea } from "../ui/textarea";
import { useChat } from "../context/chat/provider";
import { MessageType } from "@/lib/types/chat";

export default function ChatMessageInput({ roomID }: { roomID: string }) {
  const timeRef = useRef<Timer | null>(null);
  const form = useForm();
  const message = form.watch("message");
  const [typing, setTyping] = useState(false);
  const { sendMessage } = useChat();

  const handleInput = () => {
    if (timeRef.current) {
      clearTimeout(timeRef.current);
    }
    // setTyping(true);
    // if (!typing) {
    //   emitTypingEvent(roomID, true);
    // }
    // timeRef.current = setTimeout(() => {
    //   setTyping(false);
    //   emitTypingEvent(roomID, false);
    // }, 1000);
  };

  const onSumit = (data: string) => {
    sendMessage({ data: data, type: MessageType.Text, room_id: roomID });
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
        <Textarea
          rows={1}
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
