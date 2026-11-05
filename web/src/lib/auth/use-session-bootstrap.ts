import { useEffect, useRef, useState } from "react";

import { refreshSession } from "@/lib/auth/refresh";
import { sessionStore, useSession } from "@/lib/auth/session-store";

export type SessionStatus = "loading" | "signed-in" | "signed-out";

/**
 * Restores the in-memory session after a reload. With no access token in the
 * store it calls POST /api/auth/refresh once on mount and reports "loading"
 * until that settles. Strict mode runs the effect twice; the ref guard plus
 * refreshSession's single-flight keep that to one request.
 */
export function useSessionBootstrap(): SessionStatus {
  const session = useSession();
  const [checked, setChecked] = useState(() => sessionStore.get() !== null);
  const started = useRef(false);

  useEffect(() => {
    if (started.current || sessionStore.get()) return;
    started.current = true;
    void refreshSession()
      .catch(() => null)
      .finally(() => setChecked(true));
  }, []);

  if (session) return "signed-in";
  return checked ? "signed-out" : "loading";
}
