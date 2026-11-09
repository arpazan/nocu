import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { MonitorForm } from "@/components/monitors/monitor-form";
import { createMonitor } from "@/lib/api/monitors";
import type { Monitor } from "@/lib/api/types";

const push = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push, replace: vi.fn() }),
  usePathname: () => "/monitors/new",
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("@/lib/api/monitors", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/api/monitors")>()),
  createMonitor: vi.fn(),
  updateMonitor: vi.fn(),
}));

const create = vi.mocked(createMonitor);

function renderForm(monitor?: Monitor) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false }, queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MonitorForm monitor={monitor} />
    </QueryClientProvider>,
  );
}

/** Radix Select: open the trigger labelled `label` and pick `option`. */
async function selectOption(label: string, option: string) {
  await userEvent.click(screen.getByLabelText(label));
  await userEvent.click(await screen.findByRole("option", { name: option }));
}

describe("MonitorForm", () => {
  beforeAll(() => {
    // jsdom lacks the pointer and layout APIs Radix Select relies on.
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("shows required messages without calling the API", async () => {
    renderForm();

    await userEvent.click(screen.getByRole("button", { name: "Create monitor" }));

    expect(await screen.findByText("Enter a name")).toBeInTheDocument();
    expect(screen.getByText("Enter a URL")).toBeInTheDocument();
    expect(create).not.toHaveBeenCalled();
  });

  it("rejects a timeout that is not shorter than the interval", async () => {
    renderForm();

    await userEvent.type(screen.getByLabelText("Name"), "Public website");
    await userEvent.type(screen.getByLabelText("URL"), "https://example.com");
    const timeout = screen.getByLabelText("Timeout");
    await userEvent.clear(timeout);
    await userEvent.type(timeout, "60");
    await userEvent.click(screen.getByRole("button", { name: "Create monitor" }));

    expect(await screen.findByText("Timeout must be shorter than the interval")).toBeInTheDocument();
    expect(create).not.toHaveBeenCalled();
  });

  it("creates a monitor from the minimal fields and opens it", async () => {
    create.mockResolvedValue({ id: "m-new" } as Monitor);
    renderForm();

    await userEvent.type(screen.getByLabelText("Name"), "Public website");
    await userEvent.type(screen.getByLabelText("URL"), "https://example.com/health");
    await userEvent.click(screen.getByRole("button", { name: "Create monitor" }));

    await waitFor(() => expect(create).toHaveBeenCalledTimes(1));
    expect(create).toHaveBeenCalledWith({
      type: "http",
      name: "Public website",
      url: "https://example.com/health",
      interval_seconds: 60,
      retries: 0,
      timeout_seconds: 30,
      cert_expiry_notification: false,
      domain_expiry_notification: false,
      max_redirects: 10,
      save_error_response: false,
      auto_create_incident: false,
      method: "GET",
      body_encoding: "json",
      body: "",
      headers: {},
      auth_method: "none",
      basic_username: "",
      notifications: [],
    });
    await waitFor(() => expect(push).toHaveBeenCalledWith("/monitors/m-new"));
  });

  it("asks for credentials when basic auth is chosen", async () => {
    renderForm();

    expect(screen.queryByLabelText("Username")).not.toBeInTheDocument();
    await selectOption("Auth method", "Basic auth");

    expect(await screen.findByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText("Name"), "Public website");
    await userEvent.type(screen.getByLabelText("URL"), "https://example.com");
    await userEvent.click(screen.getByRole("button", { name: "Create monitor" }));

    expect(await screen.findByText("Enter a username")).toBeInTheDocument();
    expect(screen.getByText("Enter a password")).toBeInTheDocument();
    expect(create).not.toHaveBeenCalled();
  });
});
