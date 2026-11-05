import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  MONITOR_PAGE_SIZE,
  createMonitor,
  deleteMonitor,
  heartbeatsQueryOptions,
  monitorKeys,
  monitorQueryOptions,
  monitorsQueryOptions,
  pauseMonitor,
  resumeMonitor,
  updateMonitor,
} from "@/lib/api/monitors";

const api = vi.hoisted(() => ({
  list: vi.fn(),
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/lib/api/authed", () => ({ authedCoreApi: api }));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("monitorKeys", () => {
  it("nests keys under the root so invalidating the root clears everything", () => {
    expect(monitorKeys.all).toEqual(["monitors"]);
    expect(monitorKeys.lists()).toEqual(["monitors", "list"]);
    expect(monitorKeys.list({ status: "up" })).toEqual(["monitors", "list", { status: "up" }]);
    expect(monitorKeys.details()).toEqual(["monitors", "detail"]);
    expect(monitorKeys.detail("m1")).toEqual(["monitors", "detail", "m1"]);
    expect(monitorKeys.heartbeats("m1", { important: true })).toEqual([
      "monitors",
      "detail",
      "m1",
      "heartbeats",
      { important: true },
    ]);
  });

  it("is stable for equal input and distinct for different input", () => {
    expect(monitorKeys.list({ active: true, limit: 20 })).toEqual(monitorKeys.list({ active: true, limit: 20 }));
    expect(monitorKeys.list({ active: true })).not.toEqual(monitorKeys.list({ active: false }));
    expect(monitorKeys.detail("a")).not.toEqual(monitorKeys.detail("b"));
  });
});

describe("monitorsQueryOptions", () => {
  it("lists monitors with the default page size in both the key and the request", async () => {
    api.list.mockResolvedValue({ items: [], meta: { total: 0, limit: 20, offset: 0 } });
    const options = monitorsQueryOptions({ status: "down" });

    expect(options.queryKey).toEqual(monitorKeys.list({ status: "down", limit: MONITOR_PAGE_SIZE, offset: 0 }));
    await expect(options.queryFn!({} as never)).resolves.toMatchObject({ items: [] });
    expect(api.list).toHaveBeenCalledWith("/api/v1/monitors", {
      query: { status: "down", limit: 20, offset: 0 },
    });
  });

  it("keeps an explicit limit and offset", () => {
    monitorsQueryOptions({ limit: 50, offset: 100 }).queryFn!({} as never);
    expect(api.list).toHaveBeenCalledWith("/api/v1/monitors", { query: { limit: 50, offset: 100 } });
  });
});

describe("monitorQueryOptions", () => {
  it("fetches one monitor by id", async () => {
    api.get.mockResolvedValue({ id: "m1" });
    const options = monitorQueryOptions("m1");

    expect(options.queryKey).toEqual(["monitors", "detail", "m1"]);
    await expect(options.queryFn!({} as never)).resolves.toEqual({ id: "m1" });
    expect(api.get).toHaveBeenCalledWith("/api/v1/monitors/m1");
  });
});

describe("heartbeatsQueryOptions", () => {
  it("lists heartbeats under the monitor path with filters", () => {
    const options = heartbeatsQueryOptions("m1", { important: true });
    expect(options.queryKey).toEqual(["monitors", "detail", "m1", "heartbeats", { important: true, limit: 20, offset: 0 }]);
    options.queryFn!({} as never);
    expect(api.list).toHaveBeenCalledWith("/api/v1/monitors/m1/heartbeats", {
      query: { important: true, limit: 20, offset: 0 },
    });
  });
});

describe("mutations", () => {
  it("creates with POST", async () => {
    api.post.mockResolvedValue({ id: "new" });
    await expect(createMonitor({ name: "Site", url: "https://example.com" })).resolves.toEqual({ id: "new" });
    expect(api.post).toHaveBeenCalledWith("/api/v1/monitors", { name: "Site", url: "https://example.com" });
  });

  it("updates with PATCH", async () => {
    await updateMonitor("m1", { interval_seconds: 120 });
    expect(api.patch).toHaveBeenCalledWith("/api/v1/monitors/m1", { interval_seconds: 120 });
  });

  it("pauses and resumes via the sub-resource routes", async () => {
    await pauseMonitor("m1");
    await resumeMonitor("m1");
    expect(api.post).toHaveBeenNthCalledWith(1, "/api/v1/monitors/m1/pause");
    expect(api.post).toHaveBeenNthCalledWith(2, "/api/v1/monitors/m1/resume");
  });

  it("deletes and resolves to void", async () => {
    api.delete.mockResolvedValue(undefined);
    await expect(deleteMonitor("m1")).resolves.toBeUndefined();
    expect(api.delete).toHaveBeenCalledWith("/api/v1/monitors/m1");
  });
});
