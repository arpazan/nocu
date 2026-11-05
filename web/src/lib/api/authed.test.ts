import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { withAuth } from "@/lib/api/authed";
import { ApiClient, ApiError } from "@/lib/api/client";
import { refreshSession } from "@/lib/auth/refresh";
import { sessionStore } from "@/lib/auth/session-store";

vi.mock("@/lib/auth/refresh", () => ({ refreshSession: vi.fn() }));

const refresh = vi.mocked(refreshSession);

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function authHeader(init: RequestInit | undefined): string | undefined {
  return (init?.headers as Record<string, string> | undefined)?.Authorization;
}

describe("withAuth", () => {
  beforeEach(() => {
    sessionStore.set({ access_token: "old", token_type: "Bearer", expires_in: 900 });
  });

  afterEach(() => {
    sessionStore.clear();
    vi.clearAllMocks();
  });

  it("injects the current access token as a bearer header", async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => jsonResponse(200, { data: { id: "u1" } }));
    const client = withAuth(new ApiClient("http://identity.test", fetchImpl));

    await expect(client.get<{ id: string }>("/api/v1/me")).resolves.toEqual({ id: "u1" });

    expect(fetchImpl).toHaveBeenCalledTimes(1);
    expect(authHeader(fetchImpl.mock.calls[0][1])).toBe("Bearer old");
    expect(refresh).not.toHaveBeenCalled();
  });

  it("leaves an explicit token alone", async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => jsonResponse(200, { data: null }));
    const client = withAuth(new ApiClient("http://identity.test", fetchImpl));

    await client.get("/api/v1/me", { token: "explicit" });
    await client.get("/healthz", { token: null });

    expect(authHeader(fetchImpl.mock.calls[0][1])).toBe("Bearer explicit");
    expect(authHeader(fetchImpl.mock.calls[1][1])).toBeUndefined();
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    const fetchImpl = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(401, { error: { code: "unauthorized", message: "expired" } }))
      .mockResolvedValueOnce(jsonResponse(200, { data: { id: "u1" } }));
    refresh.mockResolvedValue({ access_token: "new", token_type: "Bearer", expires_in: 900 });
    const client = withAuth(new ApiClient("http://identity.test", fetchImpl));

    await expect(client.get<{ id: string }>("/api/v1/me")).resolves.toEqual({ id: "u1" });

    expect(refresh).toHaveBeenCalledTimes(1);
    expect(fetchImpl).toHaveBeenCalledTimes(2);
    expect(authHeader(fetchImpl.mock.calls[0][1])).toBe("Bearer old");
    expect(authHeader(fetchImpl.mock.calls[1][1])).toBe("Bearer new");
  });

  it("rethrows the 401 when the refresh fails", async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () =>
      jsonResponse(401, { error: { code: "unauthorized", message: "expired" } }),
    );
    refresh.mockResolvedValue(null);
    const client = withAuth(new ApiClient("http://identity.test", fetchImpl));

    const err = await client.get("/api/v1/me").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 401, code: "unauthorized" });
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it("does not refresh for other errors", async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () =>
      jsonResponse(403, { error: { code: "forbidden", message: "viewer" } }),
    );
    const client = withAuth(new ApiClient("http://core.test", fetchImpl));

    await expect(client.delete("/api/v1/monitors/1")).rejects.toMatchObject({ status: 403 });
    expect(refresh).not.toHaveBeenCalled();
  });
});
