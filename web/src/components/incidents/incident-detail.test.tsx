import { queryOptions } from "@tanstack/react-query";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { IncidentDetail } from "@/components/incidents/incident-detail";
import { makeIncident, renderWithProviders } from "@/components/incidents/test-utils";
import { ApiError } from "@/lib/api/client";
import { incidentQueryOptions, updateIncident } from "@/lib/api/incidents";
import type { Incident } from "@/lib/api/types";

const push = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
}));

let roles: string[] = ["viewer"];
vi.mock("@/lib/auth/use-me", () => ({ useRoles: () => roles }));

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast }));

vi.mock("@/lib/api/incidents", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/api/incidents")>()),
  incidentQueryOptions: vi.fn(),
  updateIncident: vi.fn(),
  deleteIncident: vi.fn(),
}));

const detailOptions = vi.mocked(incidentQueryOptions);
const update = vi.mocked(updateIncident);

function serve(result: Incident | Error) {
  detailOptions.mockImplementation((id: string) =>
    queryOptions({
      queryKey: ["incidents", "detail", id] as const,
      queryFn: () => (result instanceof Error ? Promise.reject(result) : Promise.resolve(result)),
    }),
  );
}

describe("IncidentDetail", () => {
  afterEach(() => {
    vi.clearAllMocks();
    roles = ["viewer"];
  });

  it("offers every transition from open, disabled for a viewer, and hides Delete", async () => {
    serve(makeIncident());
    renderWithProviders(<IncidentDetail id="i1" />);

    expect(await screen.findByRole("heading", { level: 1, name: "Core router down" })).toBeInTheDocument();
    for (const name of ["Acknowledge", "Resolve", "Close", "Edit"]) {
      expect(screen.getByRole("button", { name })).toBeDisabled();
    }
    expect(screen.queryByRole("button", { name: "Reopen" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    expect(screen.getByText("Opened")).toBeInTheDocument();
    expect(screen.getByText("snmp")).toBeInTheDocument();
  });

  it("acknowledges an open incident as an operator", async () => {
    roles = ["operator"];
    const incident = makeIncident();
    serve(incident);
    update.mockResolvedValue({ ...incident, status: "acknowledged", acknowledged_at: "2026-11-05T10:05:00Z" });
    renderWithProviders(<IncidentDetail id="i1" />);

    await userEvent.click(await screen.findByRole("button", { name: "Acknowledge" }));

    await waitFor(() => expect(update).toHaveBeenCalledWith("i1", { status: "acknowledged" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("Acknowledged"));
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  });

  it("shows Reopen and Delete for an admin on a resolved incident", async () => {
    roles = ["admin"];
    serve(makeIncident({ status: "resolved", resolved_at: "2026-11-05T11:00:00Z" }));
    renderWithProviders(<IncidentDetail id="i1" />);

    expect(await screen.findByRole("button", { name: "Reopen" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Close" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Delete" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Acknowledge" })).not.toBeInTheDocument();
  });

  it("explains a 409 as a stale view", async () => {
    roles = ["operator"];
    serve(makeIncident());
    update.mockRejectedValue(new ApiError(409, "conflict", "illegal transition"));
    renderWithProviders(<IncidentDetail id="i1" />);

    await userEvent.click(await screen.findByRole("button", { name: "Resolve" }));

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(
        "That change is no longer allowed; the incident was updated elsewhere.",
      ),
    );
  });

  it("renders a not-found panel on 404 instead of throwing", async () => {
    serve(new ApiError(404, "not_found", "incident not found"));
    renderWithProviders(<IncidentDetail id="missing" />);

    expect(await screen.findByText("This incident does not exist or was deleted.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to incidents" })).toHaveAttribute("href", "/incidents");
  });
});
