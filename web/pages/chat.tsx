import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Send } from "lucide-react";

export default function ChatArea() {
  return (
    <div className="flex flex-col overflow-hidden h-full">
      <div className="grow"></div>
      <div className="flex py-2 px-2 gap-2 border-t">
        <Input className="grow" />
        <Button className="" size="icon">
          <Send className="w-6 h-6" />
        </Button>
      </div>
    </div>
  );
}
