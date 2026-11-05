import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { IncidentFormDialog } from "@/components/incidents/incident-form-dialog";
import { makeIncident, polyfillPointerApis, renderWithProviders } from "@/components/incidents/test-utils";
import { ApiError } from "@/lib/api/client";
import { createIncident, updateIncident } from "@/lib/api/incidents";

const push = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/lib/api/incidents", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/api/incidents")>()),
  createIncident: vi.fn(),
  updateIncident: vi.fn(),
}));

const create = vi.mocked(createIncident);
const update = vi.mocked(updateIncident);

async function chooseSeverity(label: string) {
  await userEvent.click(screen.getByLabelText("Severity"));
  await userEvent.click(await screen.findByRole("option", { name: label }));
}

describe("IncidentFormDialog", () => {
  beforeAll(() => polyfillPointerApis());

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("shows zod messages without calling the API", async () => {
    renderWithProviders(<IncidentFormDialog open onOpenChange={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: "Create incident" }));

    expect(await screen.findByText("Enter a title")).toBeInTheDocument();
    expect(screen.getByText("Choose a severity")).toBeInTheDocument();
    expect(create).not.toHaveBeenCalled();
  });

  it("creates the incident, closes and navigates to it", async () => {
    create.mockResolvedValue(makeIncident({ id: "new-id" }));
    const onOpenChange = vi.fn();
    renderWithProviders(<IncidentFormDialog open onOpenChange={onOpenChange} />);

    await userEvent.type(screen.getByLabelText("Title"), "Core router down");
    await chooseSeverity("High");
    await userEvent.type(screen.getByLabelText("Source"), "snmp");
    await userEvent.click(screen.getByRole("button", { name: "Create incident" }));

    await waitFor(() => expect(push).toHaveBeenCalledWith("/incidents/new-id"));
    expect(create).toHaveBeenCalledWith({
      title: "Core router down",
      severity: "high",
      description: undefined,
      source: "snmp",
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("explains a 403 in terms of the missing role", async () => {
    create.mockRejectedValue(new ApiError(403, "forbidden", "forbidden"));
    renderWithProviders(<IncidentFormDialog open onOpenChange={() => {}} />);

    await userEvent.type(screen.getByLabelText("Title"), "Core router down");
    await chooseSeverity("Low");
    await userEvent.click(screen.getByRole("button", { name: "Create incident" }));

    expect(await screen.findByText("You need the operator role to do this")).toBeInTheDocument();
    expect(push).not.toHaveBeenCalled();
  });

  it("edits an existing incident without a source field and stays on the page", async () => {
    const incident = makeIncident();
    update.mockResolvedValue({ ...incident, title: "Core router flapping" });
    renderWithProviders(<IncidentFormDialog open onOpenChange={() => {}} incident={incident} />);

    expect(screen.queryByLabelText("Source")).not.toBeInTheDocument();
    const title = screen.getByLabelText("Title");
    await userEvent.clear(title);
    await userEvent.type(title, "Core router flapping");
    await userEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(update).toHaveBeenCalledWith("i1", {
        title: "Core router flapping",
        description: incident.description,
        severity: "critical",
      }),
    );
    expect(push).not.toHaveBeenCalled();
  });
});
