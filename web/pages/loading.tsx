import { Loader2 } from "lucide-react";

export default function LoadingPage() {
  return (
    <main className="flex justify-center py-4">
      <Loader2 className="w-6 h-6 animate-spin" />
    </main>
  );
}
