import { Badge } from "@/components/ui/badge";
import type { IncidentStatus } from "@/lib/api/types";
import { incidentStatusLabel } from "@/lib/incidents/transitions";

/** Outline badge for every status except open, which sits on the cream surface so it stands out in a list. */
export function IncidentStatusBadge({ status, className }: { status: IncidentStatus; className?: string }) {
  return (
    <Badge variant={status === "open" ? "secondary" : "outline"} className={className}>
      {incidentStatusLabel[status]}
    </Badge>
  );
}
