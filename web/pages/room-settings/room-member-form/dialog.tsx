import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useRoomMemberFormContext } from "./context";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { useGetUserByUsername } from "@/hooks/react-query/users";
import Alert from "@/components/alert";
import { MemberRole, Room } from "@/types/chat";
import { getMemberByUsername } from "@/utils/chat"
import { useAddRoomMember } from "@/hooks/react-query/chats";

export default function AddMemberDialog({ room }: { room: Room }) {
  const { openAddMemberDialog, setOpenAddMemberDialog } =
    useRoomMemberFormContext();
  const [username, setUsername] = useState<string>("");
  const { data: user, isLoading: isLoadingUser } =
    useGetUserByUsername(username);
  const {
    mutate: addMember,
    isPending: isAddingMember,
    error: errorAddingMember,
  } = useAddRoomMember({ roomID: room.id });

  const { canAddMember, message } = useMemo(() => {
    if (isLoadingUser) return { canAddMember: false, message: undefined };
    if (!user) {
      if (username.length > 0)
        return { canAddMember: false, message: "user not found" };
      return { canAddMember: false, message: undefined };
    }
    const member = getMemberByUsername(room, user.username);
    if (member)
      return { canAddMember: false, message: "user is already a member" };
    return {
      canAddMember: true,
    };
  }, [user, room, username, isLoadingUser]);

  const handleAddMember = async () => {
    if (!canAddMember) return;
    addMember(
      { username: user!.username, role: MemberRole.Member },
      {
        onSuccess: () => setOpenAddMemberDialog(false),
      }
    );
  };

  return (
    <Dialog open={openAddMemberDialog} onOpenChange={setOpenAddMemberDialog}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Member</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          {message && !canAddMember && <Alert message={message} />}
          {errorAddingMember && (
            <Alert variant="error" message={errorAddingMember.message} />
          )}
          <div className="space-y-2">
            <Label>Username</Label>
            <Input
              onChange={(e) => setUsername(e.target.value)}
              value={username}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            disabled={!canAddMember || isAddingMember}
            type="button"
            onClick={handleAddMember}
          >
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
