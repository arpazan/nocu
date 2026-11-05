import { queryOptions } from "@tanstack/react-query";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { IncidentsTable } from "@/components/incidents/incidents-table";
import { makeIncident, renderWithProviders } from "@/components/incidents/test-utils";
import { incidentsQueryOptions } from "@/lib/api/incidents";
import type { Incident, IncidentListFilters } from "@/lib/api/types";

const replace = vi.fn();
let search = "";
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
  usePathname: () => "/incidents",
  useSearchParams: () => new URLSearchParams(search),
}));

vi.mock("@/lib/auth/use-me", () => ({ useRoles: () => ["operator"] }));

vi.mock("@/lib/api/incidents", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/api/incidents")>()),
  incidentsQueryOptions: vi.fn(),
}));

const listOptions = vi.mocked(incidentsQueryOptions);

function serve(items: Incident[], total = items.length) {
  listOptions.mockImplementation((filters: IncidentListFilters = {}) =>
    queryOptions({
      queryKey: ["incidents", "list", filters] as const,
      queryFn: () => Promise.resolve({ items, meta: { total, limit: 20, offset: filters.offset ?? 0 } }),
    }),
  );
}

describe("IncidentsTable", () => {
  afterEach(() => {
    vi.clearAllMocks();
    search = "";
  });

  it("renders one row per incident with a link to its page", async () => {
    serve([
      makeIncident(),
      makeIncident({ id: "i2", title: "Disk nearly full", severity: "low", status: "closed", source: "" }),
    ]);
    renderWithProviders(<IncidentsTable />);

    expect(await screen.findByRole("link", { name: "Core router down" })).toHaveAttribute("href", "/incidents/i1");
    expect(screen.getByRole("link", { name: "Disk nearly full" })).toHaveAttribute("href", "/incidents/i2");
    expect(screen.getByText("Critical")).toBeInTheDocument();
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("Closed")).toBeInTheDocument();
    expect(screen.getByText("Showing 1–2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New incident" })).toBeEnabled();
  });

  it("tells the user what to do when nothing matches the filters", async () => {
    search = "status=closed";
    serve([]);
    renderWithProviders(<IncidentsTable />);

    expect(await screen.findByText(/No incidents match these filters/)).toBeInTheDocument();
    expect(listOptions).toHaveBeenCalledWith({ status: "closed", limit: 20, offset: 0 });

    await userEvent.click(screen.getByRole("button", { name: "Clear" }));
    expect(replace).toHaveBeenCalledWith("/incidents", { scroll: false });
  });

  it("shows a plain empty state without filters", async () => {
    serve([]);
    renderWithProviders(<IncidentsTable />);

    expect(await screen.findByText(/No incidents yet/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear" })).not.toBeInTheDocument();
  });
});
