import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  INCIDENT_PAGE_SIZE,
  createIncident,
  deleteIncident,
  incidentKeys,
  incidentQueryOptions,
  incidentsQueryOptions,
  updateIncident,
} from "@/lib/api/incidents";

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

describe("incidentKeys", () => {
  it("nests keys under the root", () => {
    expect(incidentKeys.all).toEqual(["incidents"]);
    expect(incidentKeys.lists()).toEqual(["incidents", "list"]);
    expect(incidentKeys.list({ status: "open" })).toEqual(["incidents", "list", { status: "open" }]);
    expect(incidentKeys.details()).toEqual(["incidents", "detail"]);
    expect(incidentKeys.detail("i1")).toEqual(["incidents", "detail", "i1"]);
  });

  it("is stable for equal input and distinct for different input", () => {
    expect(incidentKeys.list({ severity: "high" })).toEqual(incidentKeys.list({ severity: "high" }));
    expect(incidentKeys.list({ severity: "high" })).not.toEqual(incidentKeys.list({ severity: "low" }));
    expect(incidentKeys.detail("a")).not.toEqual(incidentKeys.detail("b"));
  });
});

describe("incidentsQueryOptions", () => {
  it("lists incidents with the default page size in both the key and the request", async () => {
    api.list.mockResolvedValue({ items: [], meta: { total: 0, limit: 20, offset: 0 } });
    const options = incidentsQueryOptions({ status: "open", severity: "critical" });

    expect(options.queryKey).toEqual(
      incidentKeys.list({ status: "open", severity: "critical", limit: INCIDENT_PAGE_SIZE, offset: 0 }),
    );
    await expect(options.queryFn!({} as never)).resolves.toMatchObject({ items: [] });
    expect(api.list).toHaveBeenCalledWith("/api/v1/incidents", {
      query: { status: "open", severity: "critical", limit: 20, offset: 0 },
    });
  });

  it("keeps an explicit limit and offset", () => {
    incidentsQueryOptions({ limit: 5, offset: 10 }).queryFn!({} as never);
    expect(api.list).toHaveBeenCalledWith("/api/v1/incidents", { query: { limit: 5, offset: 10 } });
  });
});

describe("incidentQueryOptions", () => {
  it("fetches one incident by id", async () => {
    api.get.mockResolvedValue({ id: "i1" });
    const options = incidentQueryOptions("i1");

    expect(options.queryKey).toEqual(["incidents", "detail", "i1"]);
    await expect(options.queryFn!({} as never)).resolves.toEqual({ id: "i1" });
    expect(api.get).toHaveBeenCalledWith("/api/v1/incidents/i1");
  });
});

describe("mutations", () => {
  it("creates with POST", async () => {
    api.post.mockResolvedValue({ id: "new", status: "open" });
    await expect(createIncident({ title: "Router down", severity: "critical" })).resolves.toMatchObject({
      status: "open",
    });
    expect(api.post).toHaveBeenCalledWith("/api/v1/incidents", { title: "Router down", severity: "critical" });
  });

  it("updates with PATCH", async () => {
    await updateIncident("i1", { status: "acknowledged" });
    expect(api.patch).toHaveBeenCalledWith("/api/v1/incidents/i1", { status: "acknowledged" });
  });

  it("deletes and resolves to void", async () => {
    api.delete.mockResolvedValue(undefined);
    await expect(deleteIncident("i1")).resolves.toBeUndefined();
    expect(api.delete).toHaveBeenCalledWith("/api/v1/incidents/i1");
  });
});
