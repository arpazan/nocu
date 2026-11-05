import { describe, expect, it } from "vitest";

import { WRITE_ROLES, canDelete, canWrite } from "@/lib/auth/roles";

describe("roles", () => {
  it("lists operator and admin as write roles", () => {
    expect(WRITE_ROLES).toEqual(["operator", "admin"]);
  });

  it("allows writes for operators and admins only", () => {
    expect(canWrite(["operator"])).toBe(true);
    expect(canWrite(["admin"])).toBe(true);
    expect(canWrite(["viewer", "admin"])).toBe(true);
    expect(canWrite(["viewer"])).toBe(false);
    expect(canWrite([])).toBe(false);
    expect(canWrite(undefined)).toBe(false);
  });

  it("allows deletes for admins only", () => {
    expect(canDelete(["admin"])).toBe(true);
    expect(canDelete(["viewer", "admin"])).toBe(true);
    expect(canDelete(["operator"])).toBe(false);
    expect(canDelete(["viewer"])).toBe(false);
    expect(canDelete(undefined)).toBe(false);
  });
});
