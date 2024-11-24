import { cn } from "@/lib/utils";

export default function Avatar({
  name,
  size = "md",
  online = false,
}: {
  name: string;
  size?: "xs" | "sm" | "md";
  online?: boolean;
}) {
  const firstChar = name.length > 0 ? name[0].toUpperCase() : "F";
  return (
    <div
      className={cn(
        "bg-muted bg-gray-300 rounded-full flex items-center justify-center font-medium",
        online && "border-green-400 border-2",
        size === "xs" && "h-4 w-4 text-xs border",
        size === "sm" && "h-7 w-7 text-sm",
        size === "md" && "h-9 w-9"
      )}
    >
      {firstChar}
    </div>
  );
}
