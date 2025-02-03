import { Button } from "@/components/ui/button";
import { Send } from "lucide-react";
import { useForm } from "react-hook-form";
import { Form, FormField } from "../ui/form";
import { useRef } from "react";
import { Textarea } from "../ui/textarea";
import { useChat } from "../context/chat/provider";
import { MessageType } from "@/lib/types/chat";

export default function ChatMessageInput({ roomID }: { roomID: string }) {
  const timeRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const form = useForm<{ message: string }>();
  const message = form.watch("message");
  const { sendMessage } = useChat();

  const handleInput = (e: React.FormEvent<HTMLTextAreaElement>) => {
    if (textareaRef.current) {
      // Reset height to compute the new scrollHeight correctly
      textareaRef.current.style.height = "auto";

      const maxHeight = 100;
      const newHeight = Math.min(textareaRef.current.scrollHeight, maxHeight);

      textareaRef.current.style.height = `${newHeight}px`;
      // If content exceeds the max height, show a scrollbar
      textareaRef.current.style.overflowY =
        textareaRef.current.scrollHeight > maxHeight ? "auto" : "hidden";
    }
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
    sendMessage({ data, type: MessageType.Text, room_id: roomID });
    form.reset({ message: "" });

    // Reset the height back to auto after sending
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  };

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
              onChange={field.onChange}
              ref={textareaRef}
              rows={1}
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
