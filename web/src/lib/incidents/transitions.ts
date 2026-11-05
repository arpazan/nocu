import type { IncidentStatus, Severity } from "@/lib/api/types";

/**
 * Mirrors shared/model/incident.go IncidentStatus.CanTransitionTo:
 * open -> acknowledged -> resolved -> closed, any non-closed status may jump
 * straight to closed, resolved may reopen to open. The server also lets open
 * skip straight to resolved. Order is forward steps, then reopen, then close.
 */
const NEXT: Record<IncidentStatus, readonly IncidentStatus[]> = {
  open: ["acknowledged", "resolved", "closed"],
  acknowledged: ["resolved", "closed"],
  resolved: ["open", "closed"],
  closed: [],
};

/** Statuses the incident may move to from `current`, in display order. */
export function nextStatuses(current: IncidentStatus): IncidentStatus[] {
  return [...NEXT[current]];
}

/** True when the server would accept a `from` -> `to` status change. */
export function canTransition(from: IncidentStatus, to: IncidentStatus): boolean {
  return NEXT[from].includes(to);
}

/** Every status, in lifecycle order. */
export const INCIDENT_STATUSES: readonly IncidentStatus[] = ["open", "acknowledged", "resolved", "closed"];

/** Every severity, most urgent first. */
export const SEVERITIES: readonly Severity[] = ["critical", "high", "medium", "low"];

export const incidentStatusLabel: Record<IncidentStatus, string> = {
  open: "Open",
  acknowledged: "Acknowledged",
  resolved: "Resolved",
  closed: "Closed",
};

export const severityLabel: Record<Severity, string> = {
  critical: "Critical",
  high: "High",
  medium: "Medium",
  low: "Low",
};

/** Button verb for moving an incident TO the given status. */
export const transitionVerb: Record<IncidentStatus, string> = {
  open: "Reopen",
  acknowledged: "Acknowledge",
  resolved: "Resolve",
  closed: "Close",
};
