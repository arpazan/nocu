import { describe, expect, it } from "vitest";

import { createIncidentSchema, updateIncidentSchema } from "@/lib/incidents/schemas";

function issues(result: { success: boolean; error?: { issues: { path: PropertyKey[]; message: string }[] } }) {
  return result.error?.issues.map((i) => [i.path.join("."), i.message]);
}

describe("createIncidentSchema", () => {
  it("accepts the Bruno example and trims the title", () => {
    const result = createIncidentSchema.safeParse({
      title: "  Core router down ",
      description: "BGP sessions dropped on edge-01",
      severity: "critical",
      source: "snmp",
    });
    expect(result.success).toBe(true);
    expect(result.data?.title).toBe("Core router down");
  });

  it("accepts the minimal body", () => {
    expect(createIncidentSchema.safeParse({ title: "x", severity: "low" }).success).toBe(true);
  });

  it("rejects a blank title and a missing severity with field messages", () => {
    expect(issues(createIncidentSchema.safeParse({ title: "   ", severity: "" }))).toEqual([
      ["title", "Enter a title"],
      ["severity", "Choose a severity"],
    ]);
  });

  it("rejects an unknown severity", () => {
    expect(issues(createIncidentSchema.safeParse({ title: "x", severity: "urgent" }))).toEqual([
      ["severity", "Choose a severity"],
    ]);
  });

  it("enforces the column sizes", () => {
    expect(issues(createIncidentSchema.safeParse({ title: "a".repeat(256), severity: "low", source: "s".repeat(129) }))).toEqual([
      ["title", "Title must be 255 characters or fewer"],
      ["source", "Source must be 128 characters or fewer"],
    ]);
  });
});

describe("updateIncidentSchema", () => {
  it("accepts an empty patch and a status-only patch", () => {
    expect(updateIncidentSchema.safeParse({}).success).toBe(true);
    expect(updateIncidentSchema.safeParse({ status: "acknowledged" }).success).toBe(true);
  });

  it("rejects an empty title and unknown status", () => {
    expect(issues(updateIncidentSchema.safeParse({ title: "", status: "done" }))).toEqual([
      ["title", "Enter a title"],
      ["status", "Choose a status"],
    ]);
  });
});
