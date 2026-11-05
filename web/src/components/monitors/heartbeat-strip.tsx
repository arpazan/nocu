import { cn } from "cn";

import type { Heartbeat } from "@/lib/api/types";
import { formatDateTime, formatLatency } from "@/lib/format";

const MIN_PX = 4;
const MAX_PX = 24;

/** Bar colour per probe status; empty slots use the muted surface. */
const BAR_COLOR: Record<Heartbeat["status"], string> = {
  up: "bg-tertiary",
  down: "bg-primary",
  pending: "bg-muted",
};

export interface HeartbeatStripProps {
  /** Newest first, as the API returns them. Only the first `count` are shown. */
  heartbeats: Heartbeat[];
  /** How many slots to draw; missing slots on the left stay empty. */
  count?: number;
  size?: "sm" | "lg";
  className?: string;
}

/** Bar height in px scaled to latency relative to the slowest probe in the set. */
function barHeight(latencyMs: number, maxLatency: number, scale: number): number {
  const min = MIN_PX * scale;
  const max = MAX_PX * scale;
  if (maxLatency <= 0 || !Number.isFinite(latencyMs) || latencyMs <= 0) return min;
  return Math.round(min + (max - min) * Math.min(latencyMs / maxLatency, 1));
}

function summarize(shown: Heartbeat[], count: number): string {
  if (shown.length === 0) return `No checks yet (0 of ${count})`;
  const up = shown.filter((h) => h.status === "up").length;
  return `${up} of ${shown.length} checks up`;
}

/**
 * The last N probes as thin vertical bars, newest on the right. Height tracks
 * latency; colour tracks status. Each bar carries a title with the probe
 * details, and the whole strip is an image with a plain-language summary.
 */
export function HeartbeatStrip({ heartbeats, count = 30, size = "sm", className }: HeartbeatStripProps) {
  const scale = size === "lg" ? 2 : 1;
  // Oldest on the left, newest on the right.
  const shown = heartbeats.slice(0, count).reverse();
  const maxLatency = shown.reduce((max, h) => Math.max(max, h.latency_ms), 0);
  const empty = Math.max(count - shown.length, 0);

  return (
    <div
      role="img"
      aria-label={summarize(shown, count)}
      className={cn("flex items-end", size === "lg" ? "h-12 gap-1" : "h-6 gap-0.5", className)}
      style={{ minHeight: MAX_PX * scale }}
    >
      {Array.from({ length: empty }, (_, i) => (
        <span
          key={`empty-${i}`}
          data-slot="heartbeat-empty"
          className={cn("rounded-sm bg-muted", size === "lg" ? "w-2" : "w-1")}
          style={{ height: MIN_PX * scale }}
        />
      ))}
      {shown.map((h) => (
        <span
          key={h.id}
          data-slot="heartbeat-bar"
          data-status={h.status}
          title={`${formatDateTime(h.checked_at)} · ${formatLatency(h.latency_ms)}${h.message ? ` · ${h.message}` : ""}`}
          className={cn("rounded-sm", size === "lg" ? "w-2" : "w-1", BAR_COLOR[h.status])}
          style={{ height: barHeight(h.latency_ms, maxLatency, scale) }}
        />
      ))}
    </div>
  );
}
