import { NextResponse } from "next/server";

import type { TokenPair } from "@/lib/api/types";
import {
  clearRefreshCookie,
  errorResponse,
  identityServer,
  readRefreshToken,
  setRefreshCookie,
  toAccessSession,
} from "@/lib/auth/session";

/** POST /api/auth/refresh: rotates the refresh cookie and returns a new access token. */
export async function POST() {
  const refreshToken = await readRefreshToken();
  if (!refreshToken) {
    return NextResponse.json({ error: { code: "unauthorized", message: "no session" } }, { status: 401 });
  }
  try {
    const pair = await identityServer.post<TokenPair>("/api/v1/auth/refresh", { refresh_token: refreshToken });
    const res = NextResponse.json({ data: toAccessSession(pair) });
    setRefreshCookie(res, pair.refresh_token);
    return res;
  } catch (err) {
    const res = errorResponse(err);
    if (res.status === 401) clearRefreshCookie(res);
    return res;
  }
}
