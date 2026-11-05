import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Overview } from "@/components/overview/overview";
import type { Incident, Monitor } from "@/lib/api/types";
import type { RealtimeEvent } from "@/lib/realtime/events";

const fixtures = vi.hoisted(() => ({
  monitors: [] as unknown[],
  incidents: [] as unknown[],
  events: [] as unknown[],
  connection: "open" as string,
  fail: null as Error | null,
}));

vi.mock("@/lib/api/monitors", () => ({
  monitorsQueryOptions: (filters: unknown) => ({
    queryKey: ["monitors", "list", filters],
    queryFn: async () => {
      if (fixtures.fail) throw fixtures.fail;
      return { items: fixtures.monitors, meta: { total: fixtures.monitors.length, limit: 200, offset: 0 } };
    },
  }),
}));

vi.mock("@/lib/api/incidents", () => ({
  incidentsQueryOptions: (filters: unknown) => ({
    queryKey: ["incidents", "list", filters],
    queryFn: async () => ({
      items: fixtures.incidents,
      meta: { total: fixtures.incidents.length, limit: 200, offset: 0 },
    }),
  }),
}));

vi.mock("@/lib/realtime/event-store", () => ({
  useRealtimeEvents: () => fixtures.events,
  useRealtimeConnection: () => fixtures.connection,
}));

vi.mock("@/components/backend-health", () => ({
  BackendHealth: () => <div data-testid="backend-health" />,
}));

function monitor(overrides: Partial<Monitor>): Monitor {
  return {
    id: "m1",
    type: "http",
    name: "Public website",
    url: "https://example.com",
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
    failure_count: 0,
    notifications: [],
    created_at: "2026-11-05T09:00:00Z",
    updated_at: "2026-11-05T09:00:00Z",
    ...overrides,
  };
}

function incident(overrides: Partial<Incident>): Incident {
  return {
    id: "i1",
    title: "Checkout latency",
    description: "",
    severity: "high",
    status: "open",
    source: "manual",
    created_at: "2026-11-05T09:00:00Z",
    updated_at: "2026-11-05T09:00:00Z",
    ...overrides,
  };
}

function renderOverview() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Overview />
    </QueryClientProvider>,
  );
}

describe("Overview", () => {
  afterEach(() => {
    fixtures.monitors = [];
    fixtures.incidents = [];
    fixtures.events = [];
    fixtures.connection = "open";
    fixtures.fail = null;
  });

  it("shows a loading skeleton, then the situation sentence and the empty attention copy", async () => {
    fixtures.monitors = [monitor({ id: "m1" }), monitor({ id: "m2", name: "API" }), monitor({ id: "m3", name: "Docs" })];
    fixtures.incidents = [incident({ status: "resolved" })];
    renderOverview();

    expect(screen.getByLabelText("Loading overview")).toBeInTheDocument();

    expect(await screen.findByRole("heading", { level: 1 })).toHaveTextContent(
      "All 3 monitors are up and no incidents are open.",
    );
    expect(screen.getByText("Live")).toBeInTheDocument();
    expect(screen.getByText("Nothing needs attention right now.")).toBeInTheDocument();
    expect(screen.getByText("Events appear here as they happen.")).toBeInTheDocument();
    expect(screen.getByTestId("backend-health")).toBeInTheDocument();
  });

  it("highlights the down count and lists what needs attention, critical first", async () => {
    fixtures.monitors = [
      monitor({ id: "m1", name: "Public website", last_status: "down", last_checked_at: "2026-11-05T09:59:00Z" }),
      monitor({ id: "m2", name: "API" }),
    ];
    fixtures.incidents = [
      incident({ id: "i1", title: "Minor glitch", severity: "low" }),
      incident({ id: "i2", title: "Database down", severity: "critical", status: "acknowledged" }),
      incident({ id: "i3", title: "Old one", status: "closed" }),
    ];
    fixtures.connection = "closed";
    renderOverview();

    const heading = await screen.findByRole("heading", { level: 1 });
    expect(heading).toHaveTextContent("1 of 2 monitors is down. 2 incidents are open.");
    expect(heading.querySelector(".text-primary")).toHaveTextContent("1");
    expect(screen.getByText("Reconnecting…")).toBeInTheDocument();

    const links = screen.getAllByRole("link");
    expect(links.map((l) => l.getAttribute("href"))).toEqual(["/monitors/m1", "/incidents/i2", "/incidents/i1"]);
    expect(links[0]).toHaveTextContent("Public website");
    expect(links[0]).toHaveTextContent("https://example.com");
    expect(links[1]).toHaveTextContent("Database down");
    expect(links[1]).toHaveTextContent("Acknowledged");
    expect(screen.queryByText("Old one")).not.toBeInTheDocument();
  });

  it("renders live events and hides quiet ones", async () => {
    const base = { occurred_at: "2026-11-05T10:00:00Z", source: "scheduler", received_at: new Date().toISOString() };
    fixtures.events = [
      { id: "e1", subject: "job.heartbeat", data: {}, ...base },
      {
        id: "e2",
        subject: "monitor.down",
        data: { monitor_id: "m1", name: "API", url: "https://api", status: "down", message: "HTTP 503", http_status: 503, latency_ms: 10, checked_at: "t" },
        ...base,
      },
    ] satisfies RealtimeEvent[];
    renderOverview();

    expect(await screen.findByText("API is down")).toBeInTheDocument();
    expect(screen.getByText("HTTP 503")).toBeInTheDocument();
    expect(screen.getByText("just now")).toBeInTheDocument();
    expect(screen.queryByText("Scheduler ran a job")).not.toBeInTheDocument();
  });

  it("shows the error inline when a query fails", async () => {
    fixtures.fail = new Error("core-api unreachable");
    renderOverview();

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load the overview: core-api unreachable");
  });
});
