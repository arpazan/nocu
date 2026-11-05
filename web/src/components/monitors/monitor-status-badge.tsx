import { Badge } from "@/components/ui/badge";
import type { Monitor } from "@/lib/api/types";
import { monitorStatusLabel } from "@/lib/monitors/labels";

/**
 * Status badges for one monitor: the probe status (up in tertiary blue, down
 * in orange, pending on the cream surface) and, when checks are paused, an
 * outline "Paused" badge next to it.
 */
export function MonitorStatusBadge({ monitor }: { monitor: Pick<Monitor, "last_status" | "active"> }) {
  const variant = monitor.last_status === "up" ? "tertiary" : monitor.last_status === "down" ? "default" : "secondary";
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <Badge variant={variant}>{monitorStatusLabel[monitor.last_status]}</Badge>
      {!monitor.active ? <Badge variant="outline">Paused</Badge> : null}
    </span>
  );
}
