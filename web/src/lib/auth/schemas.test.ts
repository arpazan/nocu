import { describe, expect, it } from "vitest";

import { loginSchema, registerSchema } from "@/lib/auth/schemas";

describe("loginSchema", () => {
  it("accepts a valid login", () => {
    expect(loginSchema.safeParse({ email: "ops@example.com", password: "correct horse" }).success).toBe(true);
  });

  it("rejects a malformed email and a short password with field messages", () => {
    const result = loginSchema.safeParse({ email: "nope", password: "short" });
    expect(result.success).toBe(false);
    const issues = result.error?.issues.map((i) => [i.path.join("."), i.message]);
    expect(issues).toEqual([
      ["email", "Enter a valid email address"],
      ["password", "Password must be at least 8 characters"],
    ]);
  });
});

describe("registerSchema", () => {
  it("accepts a registration with or without a name", () => {
    expect(registerSchema.safeParse({ email: "ops@example.com", password: "correct horse" }).success).toBe(true);
    expect(
      registerSchema.safeParse({ email: "ops@example.com", password: "correct horse", name: "Ops" }).success,
    ).toBe(true);
  });

  it("trims the name", () => {
    const result = registerSchema.safeParse({ email: "ops@example.com", password: "correct horse", name: "  Ops " });
    expect(result.success && result.data.name).toBe("Ops");
  });

  it("rejects an over-long name alongside the login messages", () => {
    const result = registerSchema.safeParse({ email: "nope", password: "short", name: "x".repeat(121) });
    expect(result.success).toBe(false);
    const issues = result.error?.issues.map((i) => [i.path.join("."), i.message]);
    expect(issues).toEqual([
      ["email", "Enter a valid email address"],
      ["password", "Password must be at least 8 characters"],
      ["name", "Name must be at most 120 characters"],
    ]);
  });
});
