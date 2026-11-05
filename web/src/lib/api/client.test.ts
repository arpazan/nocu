import { describe, expect, it, vi } from "vitest";

import { ApiClient, ApiError } from "@/lib/api/client";

function mockFetch(status: number, body: unknown, statusText = "") {
  return vi.fn<typeof fetch>(async () =>
    new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      statusText,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

describe("ApiClient", () => {
  it("unwraps the data envelope", async () => {
    const fetchImpl = mockFetch(200, { data: { id: "1", title: "Router down" } });
    const client = new ApiClient("http://api.test", fetchImpl);

    await expect(client.get<{ id: string }>("/api/v1/incidents/1")).resolves.toEqual({
      id: "1",
      title: "Router down",
    });
  });

  it("returns items and meta for lists, with a fallback meta", async () => {
    const withMeta = new ApiClient(
      "http://api.test",
      mockFetch(200, { data: [{ id: "1" }], meta: { total: 10, limit: 1, offset: 0 } }),
    );
    await expect(withMeta.list("/api/v1/incidents")).resolves.toEqual({
      items: [{ id: "1" }],
      meta: { total: 10, limit: 1, offset: 0 },
    });

    const withoutMeta = new ApiClient("http://api.test", mockFetch(200, { data: [{ id: "1" }] }));
    await expect(withoutMeta.list("/api/v1/incidents")).resolves.toMatchObject({
      meta: { total: 1, limit: 1, offset: 0 },
    });
  });

  it("sends JSON bodies, bearer tokens and query params", async () => {
    const fetchImpl = mockFetch(201, { data: { ok: true } });
    const client = new ApiClient("http://api.test/", fetchImpl);

    await client.post("/api/v1/incidents", { title: "x" }, { token: "abc", query: { limit: 5, q: undefined } });

    const [url, init] = fetchImpl.mock.calls[0];
    expect(url).toBe("http://api.test/api/v1/incidents?limit=5");
    expect(init?.method).toBe("POST");
    expect(init?.body).toBe(JSON.stringify({ title: "x" }));
    expect(init?.headers).toMatchObject({
      Authorization: "Bearer abc",
      "Content-Type": "application/json",
    });
  });

  it("converts the backend error envelope into ApiError", async () => {
    const client = new ApiClient(
      "http://api.test",
      mockFetch(401, { error: { code: "unauthorized", message: "invalid or expired token" } }),
    );

    const err = await client.get("/api/v1/me").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 401, code: "unauthorized", message: "invalid or expired token" });
    expect((err as ApiError).isUnauthorized).toBe(true);
  });

  it("falls back to a generic ApiError when the body is not an envelope", async () => {
    const client = new ApiClient("http://api.test", mockFetch(502, undefined, "Bad Gateway"));

    await expect(client.get("/healthz")).rejects.toMatchObject({
      status: 502,
      code: "http_error",
      message: "Bad Gateway",
    });
  });

  it("resolves to undefined for 204 responses", async () => {
    const client = new ApiClient("http://api.test", mockFetch(204, undefined));
    await expect(client.delete("/api/v1/incidents/1")).resolves.toBeUndefined();
  });
});
