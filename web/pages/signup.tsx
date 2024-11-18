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
import { UserSignupRequest, UserSignupRequestSchema } from "@/types/user";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@/components/alert";
import { Link, useNavigate } from "react-router-dom";
import { useSignup } from "@/hooks/users";

export default function Signup() {
  const form = useForm<UserSignupRequest>({
    resolver: zodResolver(UserSignupRequestSchema),
  });
  const navigate = useNavigate();

  const { trigger, isMutating, error } = useSignup();

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
            onSubmit={form.handleSubmit((data) => {
              trigger({ ...data }, { onSuccess: () => navigate("/signin") });
            })}
            className="grid gap-4"
          >
            <FormField
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input {...field} type="text" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
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
              Sign up
            </Button>
          </form>
        </Form>

        <p className="text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link className="underline" to="/signin">
            Sign in
          </Link>
        </p>
      </div>
    </main>
  );
}
