import { Room, RoomMember } from "@/lib/types/chat";
import {
  Table as _Table,
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
import { useRealtimeUserInfo } from "@/real-time/hooks";
import { cn } from "@/lib/utils";

export default function Table({ room }: { room: Room }) {
  const { setOpenAddMemberDialog } = useRoomMemberFormContext();
  const { get } = useRealtimeUserInfo();
  return (
    <div className="grid gap-4">
      <_Table>
        <TableHeader className="">
          <TableRow>
            <TableHead>Username</TableHead>
            <TableHead>Status</TableHead>
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
                <div className="flex">
                  <div
                    className={cn(
                      "px-2 py-1 rounded-md text-xs font-medium",
                      get(member.username).online
                        ? "bg-green-200 text-green-600 border-green-600"
                        : "bg-red-200 border-red-600 text-red-600"
                    )}
                  >
                    {get(member.username).online ? "online" : "offline"}
                  </div>
                </div>
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
      </_Table>
      <div className="flex justify-end">
        <Button onClick={() => setOpenAddMemberDialog(true)}>Add Member</Button>
      </div>
    </div>
  );
}

function RoomMemberActionsDropdownMenu({ member }: { member: RoomMember }) {
  const { mutate: removeMember, isPending: isRemovingMember } =
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
