import { Room, RoomMember } from "@/lib/types/chat";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../ui/table";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { MoreHorizontal } from "lucide-react";
import { useRoomMemberFormContext } from "./context";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useRemoveRoomMember } from "@/hooks/chats";

export default function Form({ room }: { room: Room }) {
  const { setOpenAddMemberDialog } = useRoomMemberFormContext();
  return (
    <div className="grid gap-4">
      <Table>
        <TableHeader className="">
          <TableRow>
            <TableHead>Username</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>
              <span className="sr-only">Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {room.members.map((member, idx) => (
            <TableRow key={idx}>
              <TableCell className="text-nowrap font-medium">
                {member.username}
              </TableCell>
              <TableCell>
                <Badge variant="outline">{member.role}</Badge>
              </TableCell>
              <TableCell align="right">
                <RoomMemberActionsDropdownMenu member={member} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="flex justify-end">
        <Button onClick={() => setOpenAddMemberDialog(true)}>Add Member</Button>
      </div>
    </div>
  );
}

function RoomMemberActionsDropdownMenu({ member }: { member: RoomMember }) {
  const { trigger: removeMember, isMutating: isRemovingMember } =
    useRemoveRoomMember({
      roomID: member.room_id,
    });
  const disabled = isRemovingMember;

  const handleRemoveMember = async () => {
    if (disabled) return;
    removeMember({ username: member.username });
  };
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon" disabled={disabled}>
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          onClick={handleRemoveMember}
          className="text-destructive focus:text-destructive hover:text-destructive"
        >
          Remove
        </DropdownMenuItem>
        <DropdownMenuItem>Edit</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
