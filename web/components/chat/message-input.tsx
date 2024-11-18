import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useSendMessage } from "@/hooks/chats";
import { MessageType } from "@/types/chat";
import { Send } from "lucide-react";
import { useForm } from "react-hook-form";
import { Form } from "../ui/form";

export default function ChatMessageInput({ roomID }: { roomID: string }) {
  const { trigger } = useSendMessage(roomID);
  const form = useForm();
  const message = form.watch("message");
  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(
          ({ message }) =>
            message &&
            trigger(
              { data: message, type: MessageType.TEXT },
              { onSuccess: () => form.reset() }
            )
        )}
        className="flex py-2 px-2 gap-2 border-t"
      >
        <Input className="grow" {...form.register("message")} />
        <Button disabled={!message} className="" size="icon" type="submit">
          <Send className="w-6 h-6" />
        </Button>
      </form>
    </Form>
  );
}
