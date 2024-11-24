import { cn } from "@/lib/utils";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./ui/dialog";
import { Input } from "./ui/input";
import {
  CircleHelp,
  CirclePlus,
  Loader2,
  MessageCirclePlus,
  X,
} from "lucide-react";
import { useState } from "react";
import { useGetUserByUsername } from "@/hooks/users";
import Alert from "./alert";
import { useFieldArray, useForm } from "react-hook-form";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "./ui/form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCreateGroupChat } from "@/hooks/chats";

const FormSchema = z.object({
  name: z.string().min(1),
  users: z
    .array(z.object({ username: z.string() }))
    .min(1, "Must have at least one member"),
});

export default function CreateGroupChatDialog() {
  const [username, setUsername] = useState("");
  const { data, error, isLoading } = useGetUserByUsername(username);
  const {
    trigger,
    error: createGroupChatError,
    isMutating: isCreatingGroupChat,
  } = useCreateGroupChat();
  const valid = Boolean(data && !isLoading);
  const form = useForm<z.infer<typeof FormSchema>>({
    resolver: zodResolver(FormSchema),
  });
  const members = useFieldArray({ control: form.control, name: "users" });

  const handleSubmit = form.handleSubmit((values) => {
    trigger({
      name: values.name,
      users: values.users.map((user) => user.username),
    });
  });

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button size="icon" variant="outline">
          <MessageCirclePlus className="w-3.5 h-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <Form {...form}>
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>Create Group Chat</DialogTitle>
              <DialogDescription>
                Search for a user to start a private chat with
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              {error && <Alert message={error.message} />}
              {createGroupChatError && (
                <Alert message={createGroupChatError.message} />
              )}
              <FormField
                name="name"
                control={form.control}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Room Name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="grid gap-3">
                <p>Memebers</p>
                {form.formState.errors.users?.message && (
                  <Alert message={form.formState.errors.users?.message} />
                )}
                <div className="flex gap-4">
                  <Input
                    className="grow"
                    type="text"
                    placeholder="Username..."
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                  />
                  <div className="flex-0">
                    <div
                      className={cn(
                        "w-9 h-9 flex items-center justify-center rounded-lg",
                        !isLoading && data && "bg-green-200 text-green-800",
                        !isLoading && !data && "bg-yellow-200 text-yellow-800",
                        isLoading && "bg-gray-200 text-gray-800",
                        valid && "cursor-pointer"
                      )}
                      onClick={() => {
                        if (!valid) return;
                        members.append({ username: data!.username });
                        setUsername("");
                      }}
                    >
                      {isLoading ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : valid ? (
                        <CirclePlus className="w-4 h-4" />
                      ) : (
                        <CircleHelp className="w-4 h-4" />
                      )}
                    </div>
                  </div>
                </div>
                <ul className="grid gap-2">
                  {members.fields.map((member, index) => (
                    <li
                      key={member.id}
                      className="px-3 py-2 border rounded-md text-sm flex items-center justify-between"
                    >
                      {member.username}
                      <Button
                        size="icon"
                        className="w-5 h-5"
                        variant="ghost"
                        onClick={() => members.remove(index)}
                      >
                        <X className="w-4 h-4" />
                      </Button>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
            <DialogFooter>
              <Button disabled={isCreatingGroupChat} type="submit">
                Chat
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
