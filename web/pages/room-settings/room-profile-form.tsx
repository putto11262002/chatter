import { Form, FormControl, FormItem, FormLabel } from "../../components/ui/form";
import { Input } from "../../components/ui/input";
import { Room } from "@/types/chat";
import { Button } from "../../components/ui/button";
import { useForm } from "react-hook-form";

export default function RoomProfileForm({ room }: { room: Room }) {
  const form = useForm();
  return (
    <Form {...form}>
      <form className="grid gap-4">
        <FormItem>
          <FormLabel>Name</FormLabel>
          <FormControl>
            <Input disabled={true} value={room.name} />
          </FormControl>
        </FormItem>
        <div className="flex justify-end">
          <Button type="submit">Save</Button>
        </div>
      </form>
    </Form>
  );
}
