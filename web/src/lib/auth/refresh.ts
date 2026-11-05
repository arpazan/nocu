import { ApiClient, ApiError } from "@/lib/api/client";
import { sessionStore } from "@/lib/auth/session-store";
import type { AccessSession } from "@/lib/auth/session-types";

/** Same-origin client for the Next.js /api/auth route handlers. */
const authApi = new ApiClient(typeof window === "undefined" ? "http://localhost" : window.location.origin);

let inFlight: Promise<AccessSession | null> | null = null;

/**
 * Exchanges the httpOnly refresh cookie for a new access token and stores it.
 * Resolves to null when the backend rejects the refresh (no cookie, expired or
 * revoked token); network failures are rethrown so callers can tell "signed
 * out" from "identity unreachable". Concurrent callers share one request, so a
 * burst of 401s from parallel queries only hits identity once.
 */
export function refreshSession(): Promise<AccessSession | null> {
  inFlight ??= authApi
    .post<AccessSession>("/api/auth/refresh")
    .then((session) => {
      sessionStore.set(session);
      return session;
    })
    .catch((err: unknown) => {
      if (err instanceof ApiError) {
        sessionStore.clear();
        return null;
      }
      throw err;
    })
    .finally(() => {
      inFlight = null;
    });
  return inFlight;
}

/**
 * Drops the refresh cookie and the in-memory access token. The caller is
 * responsible for clearing cached data and navigating to /login.
 */
export async function signOut(): Promise<void> {
  try {
    await authApi.post<void>("/api/auth/logout");
  } finally {
    sessionStore.clear();
  }
}
