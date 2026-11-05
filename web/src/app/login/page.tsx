import type { Metadata } from "next";
import { Suspense } from "react";

import { AppShell } from "@/components/app-shell";
import { LoginForm } from "@/components/login-form";

export const metadata: Metadata = { title: "Sign in" };

export default function LoginPage() {
  return (
    <AppShell variant="bare">
      {/* LoginForm reads useSearchParams, which must sit under a Suspense boundary to prerender. */}
      <Suspense>
        <LoginForm />
      </Suspense>
    </AppShell>
  );
}
