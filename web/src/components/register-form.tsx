"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/client";
import { register } from "@/lib/api/identity";
import { registerSchema, type RegisterValues } from "@/lib/auth/schemas";

export function RegisterForm() {
  const router = useRouter();

  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { email: "", password: "", name: "" },
  });

  const signUp = useMutation({
    mutationFn: (values: RegisterValues) =>
      register({ email: values.email, password: values.password, name: values.name ?? "" }),
    onSuccess: () => {
      toast.success("Account created");
      router.push("/login");
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) {
        form.setError("email", { message: "That email is already registered" });
        return;
      }
      if (err instanceof ApiError && err.status === 400) {
        form.setError("root", { message: err.message });
        return;
      }
      form.setError("root", { message: err instanceof Error ? err.message : "Registration failed" });
    },
  });

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle>Create account</CardTitle>
        <CardDescription>New accounts start with the viewer role.</CardDescription>
      </CardHeader>
      <CardContent>
        <form noValidate onSubmit={form.handleSubmit((values) => signUp.mutate(values))}>
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
                    autoComplete="new-password"
                    aria-invalid={fieldState.invalid}
                  />
                  <FieldError errors={[fieldState.error]} />
                </Field>
              )}
            />
            <Controller
              control={form.control}
              name="name"
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel htmlFor={field.name}>Name (optional)</FieldLabel>
                  <Input
                    {...field}
                    id={field.name}
                    type="text"
                    autoComplete="name"
                    aria-invalid={fieldState.invalid}
                  />
                  <FieldError errors={[fieldState.error]} />
                </Field>
              )}
            />
            <FieldError errors={[form.formState.errors.root]} />
            <Button type="submit" className="w-full" disabled={signUp.isPending}>
              {signUp.isPending ? "Creating account…" : "Create account"}
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
      <CardFooter className="justify-center text-sm text-muted-foreground">
        Already have an account?&nbsp;
        <Link href="/login" className="text-primary underline-offset-4 hover:underline">
          Sign in
        </Link>
      </CardFooter>
    </Card>
  );
}
