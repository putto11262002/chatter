import { cn } from "@/lib/utils";

export default function OnlineBadge({ size = "md" }: { size?: "sm" | "md" }) {
  return (
    <div
      className={cn(
        "rounded-full bg-green-300 border border-green-600",
        size === "sm" && "w-3 h-3",
        size === "md" && "w-4 h-4"
      )}
    />
  );
}
