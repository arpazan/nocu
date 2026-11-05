import { describe, expect, it } from "vitest";

import type { Incident } from "@/lib/api/types";
import {
  describeEvent,
  isIncidentEvent,
  isMonitorStatusEvent,
  parseFrame,
  toRealtimeEvent,
  type RealtimeEvent,
} from "@/lib/realtime/events";

function event(subject: RealtimeEvent["subject"], data: unknown): RealtimeEvent {
  return { id: "e1", subject, occurred_at: "2026-11-05T10:00:00Z", source: "test", data, received_at: "2026-11-05T10:00:01Z" };
}

const incident: Incident = {
  id: "i1",
  title: "Checkout latency",
  description: "p99 above 2s",
  severity: "high",
  status: "open",
  source: "manual",
  created_at: "2026-11-05T09:00:00Z",
  updated_at: "2026-11-05T09:00:00Z",
};

const probe = {
  monitor_id: "m1",
  name: "Public website",
  url: "https://example.com",
  status: "down",
  message: "connection refused",
  http_status: 0,
  latency_ms: 0,
  checked_at: "2026-11-05T10:00:00Z",
};

describe("parseFrame", () => {
  it("parses a hello frame", () => {
    expect(parseFrame('{"type":"hello","user_id":"u1","at":"2026-11-05T10:00:00Z"}')).toEqual({
      type: "hello",
      user_id: "u1",
      at: "2026-11-05T10:00:00Z",
    });
  });

  it("parses an event frame and keeps data opaque", () => {
    const frame = parseFrame(
      JSON.stringify({
        type: "event",
        subject: "incident.created",
        payload: { type: "incident.created", occurred_at: "t", source: "core-api", data: incident },
      }),
    );
    expect(frame).toEqual({
      type: "event",
      subject: "incident.created",
      payload: { type: "incident.created", occurred_at: "t", source: "core-api", data: incident },
    });
  });

  it("returns null for junk", () => {
    expect(parseFrame("not json")).toBeNull();
    expect(parseFrame("42")).toBeNull();
    expect(parseFrame("null")).toBeNull();
    expect(parseFrame("[]")).toBeNull();
    expect(parseFrame('{"type":"ping"}')).toBeNull();
    expect(parseFrame('{"type":"event","subject":"unknown.thing","payload":{}}')).toBeNull();
    expect(parseFrame('{"type":"event","subject":"monitor.up"}')).toBeNull();
    expect(parseFrame('{"type":"event","subject":"monitor.up","payload":"nope"}')).toBeNull();
  });

  it("fills missing payload fields with defaults", () => {
    const frame = parseFrame('{"type":"event","subject":"job.heartbeat","payload":{}}');
    expect(frame).toEqual({
      type: "event",
      subject: "job.heartbeat",
      payload: { type: "job.heartbeat", occurred_at: "", source: "", data: undefined },
    });
  });
});

describe("toRealtimeEvent", () => {
  it("stamps an id and the local receive time", () => {
    const received = new Date("2026-11-05T12:00:00Z");
    const e = toRealtimeEvent(
      { type: "event", subject: "monitor.up", payload: { type: "monitor.up", occurred_at: "", source: "s", data: probe } },
      received,
    );
    expect(e.id).toBeTruthy();
    expect(e.received_at).toBe("2026-11-05T12:00:00.000Z");
    expect(e.occurred_at).toBe("2026-11-05T12:00:00.000Z");
    expect(e.data).toBe(probe);
  });

  it("gives every event a distinct id", () => {
    const frame = { type: "event", subject: "job.completed", payload: { type: "", occurred_at: "", source: "", data: null } } as const;
    expect(toRealtimeEvent(frame).id).not.toBe(toRealtimeEvent(frame).id);
  });
});

describe("type guards", () => {
  it("recognise monitor status and incident payloads", () => {
    expect(isMonitorStatusEvent(event("monitor.down", probe))).toBe(true);
    expect(isMonitorStatusEvent(event("monitor.down", { nope: true }))).toBe(false);
    expect(isMonitorStatusEvent(event("monitor.created", probe))).toBe(false);
    expect(isIncidentEvent(event("incident.updated", incident))).toBe(true);
    expect(isIncidentEvent(event("incident.deleted", { id: "i1" }))).toBe(false);
    expect(isIncidentEvent(event("incident.created", null))).toBe(false);
  });
});

describe("describeEvent", () => {
  it("monitor.down", () => {
    expect(describeEvent(event("monitor.down", probe))).toEqual({
      title: "Public website is down",
      detail: "connection refused",
      tone: "down",
    });
  });

  it("monitor.up falls back to the HTTP status when there is no message", () => {
    expect(describeEvent(event("monitor.up", { ...probe, status: "up", message: "", http_status: 200 }))).toEqual({
      title: "Public website is back up",
      detail: "HTTP 200",
      tone: "up",
    });
  });

  it("incident.created includes the severity", () => {
    expect(describeEvent(event("incident.created", incident))).toEqual({
      title: "New high incident: Checkout latency",
      detail: "p99 above 2s",
      tone: "down",
    });
    expect(describeEvent(event("incident.created", { ...incident, severity: "low", description: "" }))).toMatchObject({
      title: "New low incident: Checkout latency",
      tone: "warn",
    });
  });

  it("incident.updated reports the new status", () => {
    expect(describeEvent(event("incident.updated", { ...incident, status: "acknowledged" }))).toMatchObject({
      title: "Checkout latency is now acknowledged",
      tone: "info",
    });
    expect(describeEvent(event("incident.updated", { ...incident, status: "resolved" }))).toMatchObject({
      title: "Checkout latency is now resolved",
      tone: "up",
    });
  });

  it("incident.deleted and sla_breached", () => {
    expect(describeEvent(event("incident.deleted", { id: "i1" }))).toEqual({ title: "Incident deleted", tone: "info" });
    expect(describeEvent(event("incident.sla_breached", incident))).toMatchObject({
      title: "SLA breached: Checkout latency",
      tone: "down",
    });
  });

  it("certificate and domain expiry", () => {
    const expiry = { monitor_id: "m1", name: "Public website", url: "https://example.com", expires_at: "t", days_left: 7 };
    expect(describeEvent(event("monitor.cert_expiring", expiry))).toEqual({
      title: "Public website: certificate expires in 7 days",
      detail: "https://example.com",
      tone: "warn",
    });
    expect(describeEvent(event("monitor.domain_expiring", { ...expiry, days_left: 1 })).title).toBe(
      "Public website: domain expires in 1 day",
    );
  });

  it("marks scheduler jobs quiet", () => {
    expect(describeEvent(event("job.heartbeat", null))).toEqual({ title: "Scheduler ran a job", tone: "info", quiet: true });
    expect(describeEvent(event("job.completed", {}))).toMatchObject({ quiet: true });
    expect(describeEvent(event("job.failed", {}))).toMatchObject({ title: "Scheduler job failed", quiet: true });
  });

  it("never throws on malformed data", () => {
    expect(describeEvent(event("monitor.down", null))).toEqual({ title: "Monitor down", tone: "info" });
    expect(describeEvent(event("monitor.created", { name: "Blog" }))).toEqual({ title: "Monitor added: Blog", tone: "info" });
    expect(describeEvent(event("monitor.deleted", { id: "m1" }))).toEqual({ title: "Monitor deleted", tone: "info" });
  });
});
