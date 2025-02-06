import { Button } from "@/components/ui/button";
import { Send } from "lucide-react";
import { useForm } from "react-hook-form";
import { Form, FormField } from "../ui/form";
import { useEffect, useRef, useState } from "react";
import { Textarea } from "../ui/textarea";
import { MessageType } from "@/lib/types/chat";
import { useSendMessage, useTyping } from "@/real-time/hooks";

export default function ChatMessageInput({ roomID }: { roomID: string }) {
  const timeRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const form = useForm<{ message: string }>();
  const message = form.watch("message");
  const { send } = useSendMessage();
  const { stopTyping, startTyping } = useTyping();
  const [typing, setTyping] = useState(false);
  const [rows, setRows] = useState(1);

  const handleInput = (e: React.FormEvent<HTMLTextAreaElement>) => {
    if (!typing) {
      startTyping(roomID);
      setTyping(true);
    }

    // Reset the timeout every time the user types
    if (timeRef.current) {
      clearTimeout(timeRef.current);
    }

    timeRef.current = setTimeout(() => {
      setTyping(false);
      stopTyping(roomID);
    }, 1000);

    const newRows = Math.min(
      Math.max(e.currentTarget.value.split("\n").length, 1),
      5
    );
    setRows(newRows);
  };

  // Handle keydown to submit on Enter unless Shift is held down
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      // Trigger submission if there is a message
      form.handleSubmit(({ message }) => {
        if (message) {
          onSubmit(message);
        }
      })();
    }
  };

  const onSubmit = (data: string) => {
    data = data.trim();
    if (!data) {
      return;
    }
    send({ data, type: MessageType.Text, room_id: roomID });
    form.reset({ message: "" });

    // Reset the height back to auto after sending
    setRows(1);

    if (timeRef.current) {
      clearTimeout(timeRef.current);
    }
    if (typing) {
      setTyping(false);
      stopTyping(roomID);
    }
  };

  useEffect(() => {
    return () => {
      setTyping(false);
      form.reset({ message: "" });
      if (timeRef.current) {
        clearTimeout(timeRef.current);
      }
    };
  }, [roomID]);

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(
          ({ message }) => message && onSubmit(message)
        )}
        className="flex py-2 px-2 gap-2 border-t items-end"
      >
        <FormField
          name="message"
          render={({ field }) => (
            <Textarea
              className="grow min-h-[30px] resize-none"
              value={field.value}
              onChange={(e) => {
                field.onChange(e);
              }}
              ref={textareaRef}
              rows={rows}
              onInput={handleInput}
              onKeyDown={handleKeyDown}
            />
          )}
        />
        <Button disabled={!message} size="icon" type="submit">
          <Send className="w-6 h-6" />
        </Button>
      </form>
    </Form>
  );
}
