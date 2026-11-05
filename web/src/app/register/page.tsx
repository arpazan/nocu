import type { Metadata } from "next";

import { AppShell } from "@/components/app-shell";
import { RegisterForm } from "@/components/register-form";

export const metadata: Metadata = { title: "Create account" };

export default function RegisterPage() {
  return (
    <AppShell variant="bare">
      <RegisterForm />
    </AppShell>
  );
}
