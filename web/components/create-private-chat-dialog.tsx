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
import { CheckCircle, CircleHelp, Loader2 } from "lucide-react";
import { useState } from "react";
import { useGetUserByUsername } from "@/hooks/users";
import { useCreatePrivateChat } from "@/hooks/chats";
import Alert from "./alert";

export default function CreatePrivateChatDialog({
  children,
}: {
  children: React.ReactNode;
}) {
  const [username, setUsername] = useState("");
  const { data, error, isLoading } = useGetUserByUsername(username);
  const {
    trigger,
    isMutating,
    error: createPrivateChatError,
  } = useCreatePrivateChat();
  const valid = Boolean(data && !isLoading);
  return (
    <Dialog>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create private chat</DialogTitle>
          <DialogDescription>
            Search for a user to start a private chat with
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          {error && <Alert message={error.message} />}
          {createPrivateChatError && (
            <Alert message={createPrivateChatError.message} />
          )}
          <div className="flex gap-4 py-4">
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
                  isLoading && "bg-gray-200 text-gray-800"
                )}
              >
                {isLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : valid ? (
                  <CheckCircle className="w-4 h-4" />
                ) : (
                  <CircleHelp className="w-4 h-4" />
                )}
              </div>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={() => data && trigger({ other: data.username })}
            disabled={!valid || isMutating}
          >
            Chat
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
