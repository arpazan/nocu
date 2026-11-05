import "server-only";

import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { ApiClient, ApiError } from "@/lib/api/client";
import type { ApiErrorBody, TokenPair } from "@/lib/api/types";
import type { AccessSession } from "@/lib/auth/session-types";
import { serverEnv } from "@/lib/env.server";

/**
 * Session handling for the Next.js route handlers under /api/auth.
 *
 * The refresh token never reaches browser JavaScript: it lives in an httpOnly
 * cookie scoped to /api/auth/refresh, and the access token is returned to the
 * client to keep in memory. The Go services are untouched; this layer simply
 * proxies identity and decides where each token is stored.
 */

export const REFRESH_COOKIE = "nocu_refresh";

/** identity client for server-to-server calls. */
export const identityServer = new ApiClient(serverEnv.identityUrl);

export function toAccessSession(pair: TokenPair): AccessSession {
  return { access_token: pair.access_token, token_type: pair.token_type, expires_in: pair.expires_in };
}

export async function readRefreshToken(): Promise<string | undefined> {
  const store = await cookies();
  return store.get(REFRESH_COOKIE)?.value;
}

export function setRefreshCookie(res: NextResponse, refreshToken: string): void {
  res.cookies.set({
    name: REFRESH_COOKIE,
    value: refreshToken,
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/api/auth",
    // Matches JWT_REFRESH_TTL default (168h). identity still enforces expiry.
    maxAge: 60 * 60 * 24 * 7,
  });
}

export function clearRefreshCookie(res: NextResponse): void {
  res.cookies.set({ name: REFRESH_COOKIE, value: "", path: "/api/auth", maxAge: 0 });
}

/** Renders an ApiError (or anything else) in the same envelope the Go services use. */
export function errorResponse(err: unknown): NextResponse<ApiErrorBody> {
  if (err instanceof ApiError) {
    return NextResponse.json({ error: { code: err.code, message: err.message } }, { status: err.status });
  }
  console.error("auth route failed", err);
  return NextResponse.json(
    { error: { code: "internal_error", message: "identity service unavailable" } },
    { status: 502 },
  );
}
