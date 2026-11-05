import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HeartbeatStrip } from "@/components/monitors/heartbeat-strip";
import type { Heartbeat } from "@/lib/api/types";

function heartbeat(overrides: Partial<Heartbeat> & { id: string }): Heartbeat {
  return {
    monitor_id: "m1",
    status: "up",
    latency_ms: 100,
    http_status: 200,
    message: "",
    important: false,
    checked_at: "2026-11-05T10:00:00Z",
    created_at: "2026-11-05T10:00:00Z",
    updated_at: "2026-11-05T10:00:00Z",
    ...overrides,
  };
}

describe("HeartbeatStrip", () => {
  it("summarises the up count and colours bars by status", () => {
    const { container } = render(
      <HeartbeatStrip
        count={5}
        heartbeats={[
          heartbeat({ id: "a", status: "up", latency_ms: 50 }),
          heartbeat({ id: "b", status: "down", latency_ms: 800, message: "unexpected status 503" }),
          heartbeat({ id: "c", status: "up", latency_ms: 200 }),
        ]}
      />,
    );

    expect(screen.getByRole("img", { name: "2 of 3 checks up" })).toBeInTheDocument();

    const bars = container.querySelectorAll('[data-slot="heartbeat-bar"]');
    expect(bars).toHaveLength(3);
    // Oldest on the left, newest on the right.
    expect(bars[0]).toHaveAttribute("data-status", "up");
    expect(bars[0]).toHaveClass("bg-tertiary");
    expect(bars[1]).toHaveAttribute("data-status", "down");
    expect(bars[1]).toHaveClass("bg-primary");
    expect(bars[1]).toHaveAttribute("title", expect.stringContaining("unexpected status 503"));
    expect(bars[2]).toHaveClass("bg-tertiary");

    // Missing slots are padded on the left with muted placeholders.
    expect(container.querySelectorAll('[data-slot="heartbeat-empty"]')).toHaveLength(2);
    expect(container.querySelector('[data-slot="heartbeat-empty"]')).toHaveClass("bg-muted");
  });

  it("scales bar height to latency between 4px and 24px", () => {
    const { container } = render(
      <HeartbeatStrip
        count={2}
        heartbeats={[heartbeat({ id: "fast", latency_ms: 10 }), heartbeat({ id: "slow", latency_ms: 1000 })]}
      />,
    );
    const bars = container.querySelectorAll<HTMLElement>('[data-slot="heartbeat-bar"]');
    expect(bars[0].style.height).toBe("24px");
    expect(bars[1].style.height).toBe("4px");
  });

  it("describes an empty set", () => {
    render(<HeartbeatStrip heartbeats={[]} />);
    expect(screen.getByRole("img", { name: "No checks yet (0 of 30)" })).toBeInTheDocument();
  });
});
