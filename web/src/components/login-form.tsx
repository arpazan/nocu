"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ApiClient, ApiError } from "@/lib/api/client";
import { loginSchema, type LoginValues } from "@/lib/auth/schemas";
import { sessionStore } from "@/lib/auth/session-store";
import type { AccessSession } from "@/lib/auth/session-types";

/** Same-origin client for the Next.js /api/auth route handlers. */
const authApi = new ApiClient(typeof window === "undefined" ? "http://localhost" : window.location.origin);

/** Only same-origin paths are honoured, so `next` can never bounce to another site. */
function safeNext(value: string | null): string {
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/";
}

export function LoginForm() {
  const router = useRouter();
  const redirectTo = safeNext(useSearchParams().get("next"));

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  const login = useMutation({
    mutationFn: (values: LoginValues) => authApi.post<AccessSession>("/api/auth/login", values),
    onSuccess: (session) => {
      sessionStore.set(session);
      toast.success("Signed in");
      router.push(redirectTo);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 401) {
        form.setError("root", { message: "Email or password is incorrect" });
        return;
      }
      form.setError("root", { message: err instanceof Error ? err.message : "Sign in failed" });
    },
  });

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle>Sign in</CardTitle>
        <CardDescription>Enter the email and password you registered with.</CardDescription>
      </CardHeader>
      <CardContent>
        <form noValidate onSubmit={form.handleSubmit((values) => login.mutate(values))}>
          <FieldGroup>
            <Controller
              control={form.control}
              name="email"
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor={field.name}>Email</FieldLabel>
                  <Input
                    {...field}
                    id={field.name}
                    type="email"
                    autoFocus
                    autoComplete="email"
                    placeholder="ops@example.com"
                    aria-invalid={fieldState.invalid}
                  />
                  <FieldError errors={[fieldState.error]} />
                </Field>
              )}
            />
            <Controller
              control={form.control}
              name="password"
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor={field.name}>Password</FieldLabel>
                  <Input
                    {...field}
                    id={field.name}
                    type="password"
                    autoComplete="current-password"
                    aria-invalid={fieldState.invalid}
                  />
                  <FieldError errors={[fieldState.error]} />
                </Field>
              )}
            />
            <FieldError errors={[form.formState.errors.root]} />
            <Button type="submit" className="w-full" disabled={login.isPending}>
              {login.isPending ? "Signing in…" : "Sign in"}
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
      <CardFooter className="justify-center text-sm text-muted-foreground">
        New here?&nbsp;
        <Link href="/register" className="text-primary underline-offset-4 hover:underline">
          Create an account
        </Link>
      </CardFooter>
    </Card>
  );
}
