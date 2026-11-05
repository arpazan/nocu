import type { Incident, MonitorStatus } from "@/lib/api/types";
import { incidentStatusLabel, severityLabel } from "@/lib/incidents/transitions";

/**
 * WebSocket contract with core-api (bruno/core-api/Ws.bru). The server sends a
 * `hello` frame on connect, then one `event` frame per broker message.
 */

export type IncidentSubject = "incident.created" | "incident.updated" | "incident.deleted" | "incident.sla_breached";
export type MonitorSubject =
  | "monitor.created"
  | "monitor.updated"
  | "monitor.deleted"
  | "monitor.up"
  | "monitor.down"
  | "monitor.cert_expiring"
  | "monitor.domain_expiring";
export type JobSubject = "job.heartbeat" | "job.completed" | "job.failed";
export type EventSubject = IncidentSubject | MonitorSubject | JobSubject;

const SUBJECTS: ReadonlySet<string> = new Set<EventSubject>([
  "incident.created",
  "incident.updated",
  "incident.deleted",
  "incident.sla_breached",
  "monitor.created",
  "monitor.updated",
  "monitor.deleted",
  "monitor.up",
  "monitor.down",
  "monitor.cert_expiring",
  "monitor.domain_expiring",
  "job.heartbeat",
  "job.completed",
  "job.failed",
]);

export function isEventSubject(value: unknown): value is EventSubject {
  return typeof value === "string" && SUBJECTS.has(value);
}

export interface HelloFrame {
  type: "hello";
  user_id: string;
  at: string;
}

export interface EventPayload {
  type: string;
  occurred_at: string;
  source: string;
  data: unknown;
}

export interface EventFrame {
  type: "event";
  subject: EventSubject;
  payload: EventPayload;
}

export type ServerFrame = HelloFrame | EventFrame;

/** One event as kept in the client-side feed. */
export interface RealtimeEvent {
  id: string;
  subject: EventSubject;
  occurred_at: string;
  source: string;
  data: unknown;
  /** ISO timestamp of when this browser received the frame. */
  received_at: string;
}

/** `monitor.up` / `monitor.down` data, published by the scheduler on a status flip. */
export interface MonitorStatusEventData {
  monitor_id: string;
  name: string;
  url: string;
  status: MonitorStatus;
  message: string;
  http_status: number;
  latency_ms: number;
  checked_at: string;
}

/** `monitor.cert_expiring` / `monitor.domain_expiring` data. */
export interface MonitorExpiryEventData {
  monitor_id: string;
  name: string;
  url: string;
  expires_at: string;
  days_left: number;
}

export type MonitorStatusEvent = RealtimeEvent & { subject: "monitor.up" | "monitor.down"; data: MonitorStatusEventData };
export type MonitorExpiryEvent = RealtimeEvent & {
  subject: "monitor.cert_expiring" | "monitor.domain_expiring";
  data: MonitorExpiryEventData;
};
export type IncidentEvent = RealtimeEvent & {
  subject: "incident.created" | "incident.updated" | "incident.sla_breached";
  data: Incident;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function str(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

export function isMonitorStatusEvent(e: RealtimeEvent): e is MonitorStatusEvent {
  return (
    (e.subject === "monitor.up" || e.subject === "monitor.down") &&
    isRecord(e.data) &&
    typeof e.data.monitor_id === "string" &&
    typeof e.data.name === "string"
  );
}

export function isMonitorExpiryEvent(e: RealtimeEvent): e is MonitorExpiryEvent {
  return (
    (e.subject === "monitor.cert_expiring" || e.subject === "monitor.domain_expiring") &&
    isRecord(e.data) &&
    typeof e.data.monitor_id === "string" &&
    typeof e.data.name === "string"
  );
}

export function isIncidentEvent(e: RealtimeEvent): e is IncidentEvent {
  return (
    (e.subject === "incident.created" || e.subject === "incident.updated" || e.subject === "incident.sla_breached") &&
    isRecord(e.data) &&
    typeof e.data.id === "string" &&
    typeof e.data.title === "string" &&
    typeof e.data.severity === "string" &&
    typeof e.data.status === "string"
  );
}

/**
 * Parses one raw WebSocket message. Returns null for anything that is not a
 * well-formed hello or event frame (non-JSON, unknown type, unknown subject,
 * missing payload) so a misbehaving server can never throw inside onmessage.
 */
export function parseFrame(raw: string): ServerFrame | null {
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!isRecord(value)) return null;

  if (value.type === "hello") {
    return { type: "hello", user_id: str(value.user_id), at: str(value.at) };
  }

  if (value.type === "event") {
    if (!isEventSubject(value.subject)) return null;
    const payload = isRecord(value.payload) ? value.payload : null;
    if (!payload) return null;
    return {
      type: "event",
      subject: value.subject,
      payload: {
        type: str(payload.type, value.subject),
        occurred_at: str(payload.occurred_at),
        source: str(payload.source),
        data: payload.data,
      },
    };
  }

  return null;
}

let counter = 0;

/** Unique id for a feed entry; falls back to a counter where randomUUID is missing. */
export function nextEventId(): string {
  const c = globalThis.crypto;
  if (c && typeof c.randomUUID === "function") return c.randomUUID();
  counter += 1;
  return `evt-${counter}`;
}

/** Turns an event frame into a feed entry, stamping the local receive time. */
export function toRealtimeEvent(frame: EventFrame, receivedAt: Date = new Date()): RealtimeEvent {
  return {
    id: nextEventId(),
    subject: frame.subject,
    occurred_at: frame.payload.occurred_at || receivedAt.toISOString(),
    source: frame.payload.source,
    data: frame.payload.data,
    received_at: receivedAt.toISOString(),
  };
}

export type EventTone = "up" | "down" | "info" | "warn";

export interface EventDescription {
  title: string;
  detail?: string;
  tone: EventTone;
  /** Low-value housekeeping (scheduler jobs); feeds may hide these. */
  quiet?: boolean;
}

function days(n: number): string {
  return n === 1 ? "1 day" : `${n} days`;
}

function subjectFallback(subject: EventSubject): string {
  const [entity, action] = subject.split(".");
  return `${entity.charAt(0).toUpperCase()}${entity.slice(1)} ${action.replace(/_/g, " ")}`;
}

/** User-facing copy for one event: a short title, an optional detail line and a tone for the marker. */
export function describeEvent(e: RealtimeEvent): EventDescription {
  if (isMonitorStatusEvent(e)) {
    const detail = e.data.message || (e.data.http_status ? `HTTP ${e.data.http_status}` : undefined);
    if (e.subject === "monitor.down") return { title: `${e.data.name} is down`, detail, tone: "down" };
    return { title: `${e.data.name} is back up`, detail, tone: "up" };
  }

  if (isMonitorExpiryEvent(e)) {
    const what = e.subject === "monitor.cert_expiring" ? "certificate" : "domain";
    const left = typeof e.data.days_left === "number" ? e.data.days_left : undefined;
    return {
      title: left === undefined ? `${e.data.name}: ${what} expiring soon` : `${e.data.name}: ${what} expires in ${days(left)}`,
      detail: e.data.url,
      tone: "warn",
    };
  }

  if (isIncidentEvent(e)) {
    const sev = severityLabel[e.data.severity] ?? e.data.severity;
    const status = incidentStatusLabel[e.data.status] ?? e.data.status;
    switch (e.subject) {
      case "incident.created":
        return {
          title: `New ${sev.toLowerCase()} incident: ${e.data.title}`,
          detail: e.data.description || undefined,
          tone: e.data.severity === "critical" || e.data.severity === "high" ? "down" : "warn",
        };
      case "incident.updated":
        return {
          title: `${e.data.title} is now ${status.toLowerCase()}`,
          detail: `${sev} severity`,
          tone: e.data.status === "resolved" || e.data.status === "closed" ? "up" : "info",
        };
      case "incident.sla_breached":
        return { title: `SLA breached: ${e.data.title}`, detail: `${sev} severity, ${status.toLowerCase()}`, tone: "down" };
    }
  }

  const data = isRecord(e.data) ? e.data : {};
  switch (e.subject) {
    case "incident.deleted":
      return { title: "Incident deleted", tone: "info" };
    case "monitor.created":
      return { title: typeof data.name === "string" ? `Monitor added: ${data.name}` : "Monitor added", tone: "info" };
    case "monitor.updated":
      return { title: typeof data.name === "string" ? `Monitor updated: ${data.name}` : "Monitor updated", tone: "info" };
    case "monitor.deleted":
      return { title: "Monitor deleted", tone: "info" };
    case "job.failed":
      return { title: "Scheduler job failed", tone: "warn", quiet: true };
    case "job.heartbeat":
    case "job.completed":
      return { title: "Scheduler ran a job", tone: "info", quiet: true };
    default:
      return { title: subjectFallback(e.subject), tone: "info" };
  }
}
