import type { Incident, Monitor, Severity } from "@/lib/api/types";

/** Counts the overview sentence and attention list are built from. */
export interface Situation {
  total: number;
  up: number;
  down: number;
  pending: number;
  openIncidents: number;
}

export function summarize(monitors: readonly Monitor[], incidents: readonly Incident[]): Situation {
  let up = 0;
  let down = 0;
  let pending = 0;
  for (const m of monitors) {
    if (m.last_status === "down") down += 1;
    else if (m.last_status === "up") up += 1;
    else pending += 1;
  }
  return { total: monitors.length, up, down, pending, openIncidents: incidents.filter(isOpenIncident).length };
}

export function isOpenIncident(i: Incident): boolean {
  return i.status === "open" || i.status === "acknowledged";
}

function plural(n: number, word: string): string {
  return n === 1 ? word : `${word}s`;
}

/** "no incidents are open" / "1 incident is open" / "3 incidents are open" */
export function incidentClause(open: number): string {
  if (open === 0) return "no incidents are open";
  return `${open} ${plural(open, "incident")} ${open === 1 ? "is" : "are"} open`;
}

/**
 * The hero sentence, split so the down count can be emphasised. `lead` is the
 * text before the count, `count` the number (empty when nothing is down), and
 * `rest` the remainder including the incidents clause and the final period.
 */
export interface SituationSentence {
  lead: string;
  count: string;
  rest: string;
}

export function situationSentence(s: Situation): SituationSentence {
  const incidents = incidentClause(s.openIncidents);

  if (s.total === 0) {
    return { lead: "No monitors yet", count: "", rest: ` and ${incidents}.` };
  }

  if (s.down > 0) {
    const tail = s.openIncidents > 0 ? ` ${incidents.charAt(0).toUpperCase()}${incidents.slice(1)}.` : "";
    return {
      lead: "",
      count: String(s.down),
      rest: ` of ${s.total} ${plural(s.total, "monitor")} ${s.down === 1 ? "is" : "are"} down.${tail}`,
    };
  }

  let monitors: string;
  if (s.up === 0) {
    monitors =
      s.total === 1
        ? "The only monitor is waiting for its first check"
        : `All ${s.total} monitors are waiting for their first check`;
  } else if (s.pending > 0) {
    monitors = `${s.up} of ${s.total} monitors are up`;
  } else {
    monitors = s.total === 1 ? "The only monitor is up" : `All ${s.total} monitors are up`;
  }
  return { lead: monitors, count: "", rest: ` and ${incidents}.` };
}

/** Sort order for the attention list: most urgent first. */
export const SEVERITY_RANK: Record<Severity, number> = { critical: 0, high: 1, medium: 2, low: 3 };

/** Open and acknowledged incidents, critical first, then newest first. */
export function attentionIncidents(incidents: readonly Incident[]): Incident[] {
  return incidents
    .filter(isOpenIncident)
    .sort((a, b) => SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity] || b.created_at.localeCompare(a.created_at));
}

/** Down monitors, most recently checked first. */
export function attentionMonitors(monitors: readonly Monitor[]): Monitor[] {
  return monitors
    .filter((m) => m.last_status === "down")
    .sort((a, b) => (b.last_checked_at ?? "").localeCompare(a.last_checked_at ?? ""));
}
