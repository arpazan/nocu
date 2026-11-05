import { NextResponse } from "next/server";

import { clearRefreshCookie } from "@/lib/auth/session";

/** POST /api/auth/logout: drops the refresh cookie. Access tokens expire on their own. */
export async function POST() {
  const res = new NextResponse(null, { status: 204 });
  clearRefreshCookie(res);
  return res;
}
