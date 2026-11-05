import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import type { ReactElement } from "react";

import { TooltipProvider } from "@/components/ui/tooltip";
import type { Incident } from "@/lib/api/types";

/** Renders with the providers the (app) layout supplies: a fresh query client and tooltips. */
export function renderWithProviders(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TooltipProvider>{ui}</TooltipProvider>
    </QueryClientProvider>,
  );
}

export function makeIncident(overrides: Partial<Incident> = {}): Incident {
  return {
    id: "i1",
    title: "Core router down",
    description: "BGP sessions dropped on edge-01",
    severity: "critical",
    status: "open",
    source: "snmp",
    reporter_id: "u1",
    acknowledged_at: null,
    resolved_at: null,
    closed_at: null,
    created_at: "2026-11-05T10:00:00Z",
    updated_at: "2026-11-05T10:00:00Z",
    ...overrides,
  };
}

/** Radix Select relies on pointer-capture and scrollIntoView, which jsdom lacks. */
export function polyfillPointerApis() {
  const proto = window.HTMLElement.prototype as unknown as Record<string, unknown>;
  proto.hasPointerCapture ??= () => false;
  proto.setPointerCapture ??= () => {};
  proto.releasePointerCapture ??= () => {};
  proto.scrollIntoView ??= () => {};
}
