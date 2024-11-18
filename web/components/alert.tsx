import { cn } from "@/lib/utils";

export default function Alert({ message }: { message: string }) {
  return (
    <div
      className={cn(
        "px-3 py-2 rounded-md text-sm",
        "text-red-800 bg-red-200 border"
      )}
    >
      {message}
    </div>
  );
}
