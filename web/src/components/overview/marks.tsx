import { cn } from "cn";

import type { MonitorStatus, Severity } from "@/lib/api/types";
import type { EventTone } from "@/lib/realtime/events";

/*
 * Small colour marks local to the overview. Brand rule: orange means
 * "needs you", blue means healthy, grey means nothing to see.
 */

const severityColor: Record<Severity, string> = {
  critical: "bg-primary",
  high: "bg-primary/60",
  medium: "bg-tertiary",
  low: "bg-muted-foreground/40",
};

/** 2px vertical bar coloured by severity. */
export function SeverityBar({ severity, className }: { severity: Severity; className?: string }) {
  return (
    <span
      aria-hidden="true"
      data-severity={severity}
      className={cn("inline-block w-0.5 self-stretch rounded-full", severityColor[severity], className)}
    />
  );
}

const statusColor: Record<MonitorStatus, string> = {
  up: "bg-tertiary",
  down: "bg-primary",
  pending: "bg-muted-foreground/40",
};

/** 6px dot coloured by monitor status. */
export function StatusDot({ status, className }: { status: MonitorStatus; className?: string }) {
  return (
    <span
      aria-hidden="true"
      data-status={status}
      className={cn("inline-block size-1.5 shrink-0 rounded-full", statusColor[status], className)}
    />
  );
}

const toneColor: Record<EventTone, string> = {
  up: "bg-tertiary",
  down: "bg-primary",
  warn: "bg-primary/60",
  info: "bg-muted-foreground/40",
};

/** 6px dot coloured by event tone. */
export function ToneDot({ tone, className }: { tone: EventTone; className?: string }) {
  return (
    <span
      aria-hidden="true"
      data-tone={tone}
      className={cn("inline-block size-1.5 shrink-0 rounded-full", toneColor[tone], className)}
    />
  );
}
