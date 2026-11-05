import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { MonitorsTable } from "@/components/monitors/monitors-table";
import type { Heartbeat, Monitor } from "@/lib/api/types";

const replace = vi.fn();
let search = "";
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace, push: vi.fn() }),
  usePathname: () => "/monitors",
  useSearchParams: () => new URLSearchParams(search),
}));

let roles: string[] = ["operator"];
vi.mock("@/lib/auth/use-me", () => ({ useRoles: () => roles }));

function monitor(overrides: Partial<Monitor> & { id: string; name: string }): Monitor {
  return {
    type: "http",
    url: `https://${overrides.id}.example.com/health`,
    interval_seconds: 60,
    retries: 0,
    timeout_seconds: 30,
    active: true,
    cert_expiry_notification: false,
    domain_expiry_notification: false,
    max_redirects: 10,
    save_error_response: false,
    method: "GET",
    body_encoding: "json",
    body: "",
    headers: null,
    auth_method: "none",
    has_basic_password: false,
    has_bearer_token: false,
    last_status: "up",
    last_checked_at: new Date(Date.now() - 3 * 60_000).toISOString(),
    failure_count: 0,
    notifications: [],
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
    ...overrides,
  };
}

function heartbeat(id: string, status: Heartbeat["status"]): Heartbeat {
  return {
    id,
    monitor_id: "m1",
    status,
    latency_ms: 120,
    http_status: 200,
    message: "",
    important: false,
    checked_at: "2026-11-05T10:00:00Z",
    created_at: "2026-11-05T10:00:00Z",
    updated_at: "2026-11-05T10:00:00Z",
  };
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

/** Routes the authed client's requests: the list, then per-row heartbeats. */
function mockApi(monitors: Monitor[]) {
  return vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    if (url.includes("/heartbeats")) {
      return json({ data: [heartbeat("h1", "up"), heartbeat("h2", "down")], meta: { total: 2, limit: 30, offset: 0 } });
    }
    return json({ data: monitors, meta: { total: monitors.length, limit: 20, offset: 0 } });
  });
}

function renderTable() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MonitorsTable />
    </QueryClientProvider>,
  );
}

describe("MonitorsTable", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    replace.mockClear();
    search = "";
    roles = ["operator"];
  });

  it("renders one row per monitor with its status and heartbeats", async () => {
    mockApi([
      monitor({ id: "m1", name: "Public website" }),
      monitor({ id: "m2", name: "Staging API", last_status: "down", active: false }),
    ]);
    renderTable();

    expect(await screen.findByRole("link", { name: "Public website" })).toHaveAttribute("href", "/monitors/m1");
    expect(screen.getByRole("link", { name: "Staging API" })).toHaveAttribute("href", "/monitors/m2");
    expect(screen.getByText("https://m1.example.com/health")).toBeInTheDocument();
    expect(screen.getByText("Up")).toBeInTheDocument();
    expect(screen.getByText("Down")).toBeInTheDocument();
    expect(screen.getByText("Paused")).toBeInTheDocument();
    expect(screen.getByText("Showing 1–2 of 2")).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByRole("img", { name: "1 of 2 checks up" })).toHaveLength(2));
    expect(screen.getByRole("link", { name: "New monitor" })).toHaveAttribute("href", "/monitors/new");
  });

  it("shows the empty state with a way to add the first monitor", async () => {
    mockApi([]);
    renderTable();

    expect(await screen.findByText("No monitors yet. Add your first URL to start checking it.")).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "New monitor" }).length).toBeGreaterThan(0);
  });

  it("tells filtered views apart from an empty account", async () => {
    search = "status=down";
    mockApi([]);
    renderTable();

    expect(await screen.findByText("No monitors match these filters.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Clear" })).toBeInTheDocument();
  });

  it("disables New monitor for viewers", async () => {
    roles = ["viewer"];
    mockApi([monitor({ id: "m1", name: "Public website" })]);
    renderTable();

    await screen.findByRole("link", { name: "Public website" });
    expect(screen.queryByRole("link", { name: "New monitor" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New monitor" })).toBeDisabled();
  });
});
