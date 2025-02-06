import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Input } from "../ui/input";
import Alert from "../alert";
import { useCreateRoom } from "@/hooks/chats";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../ui/form";
import { useForm } from "react-hook-form";
import { CreateRoomPayload, createRoomSchema } from "@/lib/types/chat";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCreateRoomDialog } from "./context";
import { useNavigate } from "react-router-dom";

export default function CreateRoomDialog() {
  const { mutate, isPending, error: createRoomError } = useCreateRoom();
  const { open, setOpen } = useCreateRoomDialog();
  const navigate = useNavigate();
  const form = useForm<CreateRoomPayload>({
    resolver: zodResolver(createRoomSchema),
  });

  const handleSubmit = form.handleSubmit(async (data) => {
    mutate(data, {
      onSuccess: (res) => {
        form.reset();
        setOpen(false);
        navigate(`/${res.id}`);
      },
    });
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Room</DialogTitle>
          <DialogDescription>
            Create a new chat room where you can chat with other users.
          </DialogDescription>
        </DialogHeader>
        {createRoomError && <Alert message={createRoomError.message} />}
        <Form {...form}>
          <form onSubmit={handleSubmit}>
            <div className="py-4">
              <FormField
                name="name"
                control={form.control}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input {...field} type="text" placeholder="Name..." />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <DialogFooter>
              <Button type="submit" disabled={isPending}>
                Let's Chat!
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
