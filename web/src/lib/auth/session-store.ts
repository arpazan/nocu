import { useSyncExternalStore } from "react";

import type { AccessSession } from "@/lib/auth/session-types";

/**
 * In-memory holder for the access token. It is deliberately not persisted:
 * the refresh token lives in an httpOnly cookie, so a reload simply calls
 * POST /api/auth/refresh to get a new access token.
 */

let current: AccessSession | null = null;
const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}

export const sessionStore = {
  get: () => current,
  set(session: AccessSession | null) {
    current = session;
    emit();
  },
  clear() {
    sessionStore.set(null);
  },
  subscribe(listener: () => void) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
};

/** Current access token, or null when signed out. */
export function getAccessToken(): string | null {
  return current?.access_token ?? null;
}

/** Reactive view of the session for components. */
export function useSession(): AccessSession | null {
  return useSyncExternalStore(sessionStore.subscribe, sessionStore.get, () => null);
}
