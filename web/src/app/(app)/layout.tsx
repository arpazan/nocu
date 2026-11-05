import { Suspense } from "react";

import { AppShell } from "@/components/app-shell";
import { AuthGate } from "@/components/auth-gate";
import { RealtimeProvider } from "@/components/realtime-provider";

/**
 * The session lives only in the browser, so this group cannot be part of the static shell.
 * Opting out of instant-navigation validation silences the dev-time warning that AuthGate's
 * loading state otherwise triggers (see Next's "Authentication with Cache Components" guide).
 */
export const instant = false;

/**
 * Every route in this group requires a session; AuthGate handles the redirect.
 * AuthGate reads useSearchParams, so it sits under Suspense to keep the shell prerenderable.
 */
export default function AppLayout({ children }: LayoutProps<"/">) {
  return (
    <Suspense>
      <AuthGate>
        <RealtimeProvider>
          <AppShell>{children}</AppShell>
        </RealtimeProvider>
      </AuthGate>
    </Suspense>
  );
}
