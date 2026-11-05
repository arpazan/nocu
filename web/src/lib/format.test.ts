import { describe, expect, it } from "vitest";

import { formatDateTime, formatInterval, formatLatency, formatRelative } from "@/lib/format";

describe("formatDateTime", () => {
  it("renders day, short month, year and 24h time", () => {
    const local = new Date(2026, 10, 5, 14, 3);
    expect(formatDateTime(local.toISOString())).toBe("5 Nov 2026, 14:03");
  });

  it("returns an em dash for empty or invalid input", () => {
    expect(formatDateTime(null)).toBe("—");
    expect(formatDateTime(undefined)).toBe("—");
    expect(formatDateTime("")).toBe("—");
    expect(formatDateTime("not a date")).toBe("—");
  });
});

describe("formatRelative", () => {
  const now = new Date("2026-11-05T12:00:00Z");
  const at = (offsetSeconds: number) => new Date(now.getTime() + offsetSeconds * 1000).toISOString();

  it.each([
    [-3 * 60, "3 min ago"],
    [2 * 60, "in 2 min"],
    [-45, "45 sec ago"],
    [-2 * 3600, "2 hr ago"],
    [-3 * 86400, "3 days ago"],
    [10, "just now"],
  ])("offset %d s -> %s", (offset, expected) => {
    expect(formatRelative(at(offset), now)).toBe(expected);
  });

  it("returns an em dash for empty input", () => {
    expect(formatRelative(null, now)).toBe("—");
  });
});

describe("formatLatency", () => {
  it.each([
    [812, "812 ms"],
    [0, "0 ms"],
    [1234, "1.2 s"],
    [1000, "1.0 s"],
    [null, "—"],
  ])("%s -> %s", (ms, expected) => {
    expect(formatLatency(ms)).toBe(expected);
  });
});

describe("formatInterval", () => {
  it.each([
    [60, "60 s"],
    [90, "90 s"],
    [120, "2 min"],
    [300, "5 min"],
    [3600, "1 h"],
    [5400, "90 min"],
    [undefined, "—"],
  ])("%s -> %s", (seconds, expected) => {
    expect(formatInterval(seconds)).toBe(expected);
  });
});
