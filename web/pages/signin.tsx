import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@/components/alert";
import { Link, useNavigate } from "react-router-dom";
import { SigninPayload, signinPayloadSchema } from "@/lib/types/auth";
import { useSignin } from "@/hooks/auth";

export default function Signin() {
  const form = useForm<SigninPayload>({
    resolver: zodResolver(signinPayloadSchema),
  });

  const navigate = useNavigate();

  const { isMutating, trigger, error } = useSignin();

  return (
    <main className="h-screen w-full flex items-center justify-center">
      <div className="w-full max-w-sm border p-6 rounded-lg grid gap-4">
        <div>
          <h1 className="text-lg font-semibold">Chatter</h1>
          <h2 className="text-muted-foreground">Sign up</h2>
        </div>

        {error && <Alert message={error.message} />}
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((data) =>
              trigger(data, { onSuccess: () => navigate("/") })
            )}
            className="grid gap-4"
          >
            <FormField
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Username</FormLabel>
                  <FormControl>
                    <Input {...field} type="text" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password</FormLabel>
                  <FormControl>
                    <Input {...field} type="password" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button disabled={isMutating} type="submit">
              Sign in
            </Button>
          </form>
        </Form>
        <p className="text-sm text-muted-foreground">
          Don't have an account?{" "}
          <Link className="underline" to="/signup">
            Sign up
          </Link>
        </p>
      </div>
    </main>
  );
}
