import { useSession } from "@/components/providers/session-provider";
import { Button } from "@/components/ui/button";
import { useSignout } from "@/hooks/auth";

export default function Home() {
  const session = useSession();
  console.log(session);
  const { trigger: signout, isMutating: isSigningOut } = useSignout();
  return (
    <div className="flex flex-col overflow-hidden h-full">
      <p>Welcome back {session.name}</p>
      <Button disabled={isSigningOut} onClick={() => signout()}>
        Sign out
      </Button>
    </div>
  );
}
