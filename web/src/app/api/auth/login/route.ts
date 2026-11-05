import { NextResponse } from "next/server";

import type { LoginRequest, TokenPair } from "@/lib/api/types";
import { errorResponse, identityServer, setRefreshCookie, toAccessSession } from "@/lib/auth/session";

/** POST /api/auth/login: proxies identity login and stores the refresh token in a cookie. */
export async function POST(req: Request) {
  try {
    const body = (await req.json()) as LoginRequest;
    const pair = await identityServer.post<TokenPair>("/api/v1/auth/login", body);
    const res = NextResponse.json({ data: toAccessSession(pair) });
    setRefreshCookie(res, pair.refresh_token);
    return res;
  } catch (err) {
    return errorResponse(err);
  }
}
