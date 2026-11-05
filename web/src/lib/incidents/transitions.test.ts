import { describe, expect, it } from "vitest";

import type { IncidentStatus } from "@/lib/api/types";
import {
  INCIDENT_STATUSES,
  SEVERITIES,
  canTransition,
  incidentStatusLabel,
  nextStatuses,
  severityLabel,
  transitionVerb,
} from "@/lib/incidents/transitions";

describe("nextStatuses", () => {
  it.each<[IncidentStatus, IncidentStatus[]]>([
    ["open", ["acknowledged", "resolved", "closed"]],
    ["acknowledged", ["resolved", "closed"]],
    ["resolved", ["open", "closed"]],
    ["closed", []],
  ])("%s -> %j", (from, expected) => {
    expect(nextStatuses(from)).toEqual(expected);
  });

  it("returns a fresh array each call", () => {
    const a = nextStatuses("open");
    a.push("closed");
    expect(nextStatuses("open")).toHaveLength(3);
  });
});

describe("canTransition", () => {
  // Full table: rows are `from`, columns are `to` in lifecycle order.
  const table: Record<IncidentStatus, [boolean, boolean, boolean, boolean]> = {
    //            open   ack    res    closed
    open: [false, true, true, true],
    acknowledged: [false, false, true, true],
    resolved: [true, false, false, true],
    closed: [false, false, false, false],
  };

  for (const from of INCIDENT_STATUSES) {
    for (const [i, to] of INCIDENT_STATUSES.entries()) {
      const expected = table[from][i];
      it(`${from} -> ${to} is ${expected ? "allowed" : "rejected"}`, () => {
        expect(canTransition(from, to)).toBe(expected);
      });
    }
  }

  it("agrees with nextStatuses", () => {
    for (const from of INCIDENT_STATUSES) {
      for (const to of INCIDENT_STATUSES) {
        expect(canTransition(from, to)).toBe(nextStatuses(from).includes(to));
      }
    }
  });
});

describe("labels", () => {
  it("covers every status and severity", () => {
    expect(INCIDENT_STATUSES.map((s) => incidentStatusLabel[s])).toEqual(["Open", "Acknowledged", "Resolved", "Closed"]);
    expect(INCIDENT_STATUSES.map((s) => transitionVerb[s])).toEqual(["Reopen", "Acknowledge", "Resolve", "Close"]);
    expect(SEVERITIES.map((s) => severityLabel[s])).toEqual(["Critical", "High", "Medium", "Low"]);
  });
});
