import { describe, expect, it } from "vitest";

import type { Monitor } from "@/lib/api/types";
import {
  type MonitorFormValues,
  fromMonitor,
  fromNotification,
  headerRowsToRecord,
  monitorFormDefaults,
  monitorFormSchema,
  newNotification,
  notificationFormSchema,
  toCreateRequest,
  toNotificationInput,
  toUpdateRequest,
} from "@/lib/monitors/schemas";

function issues(result: { success: boolean; error?: { issues: { path: PropertyKey[]; message: string }[] } }) {
  return result.error?.issues.map((i) => [i.path.join("."), i.message]) ?? [];
}

function values(overrides: Partial<MonitorFormValues> = {}): MonitorFormValues {
  return { ...monitorFormDefaults, name: "Public website", url: "https://example.com/health", ...overrides };
}

/** The folder.bru example monitor. */
const stored: Monitor = {
  id: "m1",
  type: "http",
  name: "Public website",
  url: "https://example.com/health",
  interval_seconds: 60,
  retries: 1,
  timeout_seconds: 30,
  active: true,
  owner_id: "u1",
  cert_expiry_notification: true,
  domain_expiry_notification: false,
  max_redirects: 10,
  save_error_response: false,
  method: "GET",
  body_encoding: "json",
  body: "",
  headers: { "X-Probe": "nocu" },
  auth_method: "none",
  has_basic_password: false,
  has_bearer_token: false,
  last_status: "pending",
  last_checked_at: null,
  next_check_at: "2026-11-05T12:00:00Z",
  failure_count: 0,
  cert_expires_at: null,
  domain_expires_at: null,
  notifications: [
    {
      id: "n1",
      monitor_id: "m1",
      type: "discord",
      enabled: true,
      config: {
        webhook_url: "https://discord.com/api/webhooks/123/abc",
        bot_display_name: "Nocu",
        message_format: "embed",
        disable_url: false,
      },
    },
    {
      id: "n2",
      monitor_id: "m1",
      type: "webhook",
      enabled: true,
      config: {
        post_url: "https://hooks.example.com/nocu",
        http_method: "POST",
        request_body: "json",
        additional_headers: { "X-Token": "secret" },
      },
    },
  ],
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

describe("monitorFormSchema", () => {
  it("accepts the defaults with a name and url", () => {
    expect(issues(monitorFormSchema.safeParse(values()))).toEqual([]);
  });

  it("requires a name and an http(s) url", () => {
    expect(issues(monitorFormSchema.safeParse(values({ name: "  ", url: "ftp://example.com" })))).toEqual([
      ["name", "Enter a name"],
      ["url", "Enter an http or https URL"],
    ]);
    expect(issues(monitorFormSchema.safeParse(values({ url: "example.com" })))).toEqual([
      ["url", "Enter an http or https URL"],
    ]);
    expect(issues(monitorFormSchema.safeParse(values({ url: "http://localhost:8080" })))).toEqual([]);
  });

  it("enforces the numeric minimums", () => {
    expect(
      issues(monitorFormSchema.safeParse(values({ interval_seconds: 19, retries: -1, timeout_seconds: 0, max_redirects: -1 }))),
    ).toEqual([
      ["interval_seconds", "At least 20 seconds"],
      ["retries", "Cannot be negative"],
      ["timeout_seconds", "At least 1 second"],
      ["max_redirects", "Cannot be negative"],
    ]);
  });

  it("rejects fractional and missing numbers", () => {
    expect(issues(monitorFormSchema.safeParse(values({ interval_seconds: 60.5 })))).toEqual([
      ["interval_seconds", "Enter a whole number"],
    ]);
    expect(issues(monitorFormSchema.safeParse(values({ timeout_seconds: Number.NaN })))).toEqual([
      ["timeout_seconds", "Enter a timeout"],
    ]);
  });

  it("requires the timeout to be shorter than the interval", () => {
    expect(issues(monitorFormSchema.safeParse(values({ interval_seconds: 30, timeout_seconds: 30 })))).toEqual([
      ["timeout_seconds", "Timeout must be shorter than the interval"],
    ]);
    expect(issues(monitorFormSchema.safeParse(values({ interval_seconds: 30, timeout_seconds: 29 })))).toEqual([]);
  });

  it("rejects unknown enum values", () => {
    const result = monitorFormSchema.safeParse(
      values({ method: "TRACE" as never, body_encoding: "yaml" as never, auth_method: "digest" as never }),
    );
    expect(issues(result)).toEqual([
      ["method", "Choose a method"],
      ["body_encoding", "Choose an encoding"],
      ["auth_method", "Choose an auth method"],
    ]);
  });

  it("flags header rows with a value but no key, and ignores blank rows", () => {
    const result = monitorFormSchema.safeParse(
      values({ headers: [{ key: "", value: "" }, { key: "  ", value: "x" }, { key: "X-Probe", value: "" }] }),
    );
    expect(issues(result)).toEqual([["headers.1.key", "Enter a header name"]]);
  });

  describe("auth_method basic", () => {
    it("requires username and password on create", () => {
      expect(issues(monitorFormSchema.safeParse(values({ auth_method: "basic" })))).toEqual([
        ["basic_username", "Enter a username"],
        ["basic_password", "Enter a password"],
      ]);
    });

    it("accepts typed credentials", () => {
      expect(
        issues(monitorFormSchema.safeParse(values({ auth_method: "basic", basic_username: "u", basic_password: "p" }))),
      ).toEqual([]);
    });

    it("accepts a stored password in edit mode", () => {
      expect(
        issues(monitorFormSchema.safeParse(values({ auth_method: "basic", basic_username: "u", has_basic_password: true }))),
      ).toEqual([]);
    });

    it("rejects clearing the stored password while basic auth stays selected", () => {
      expect(
        issues(
          monitorFormSchema.safeParse(
            values({ auth_method: "basic", basic_username: "u", has_basic_password: true, clear_basic_password: true }),
          ),
        ),
      ).toEqual([["basic_password", "Enter a password"]]);
    });

    it("does not require credentials when auth is none", () => {
      expect(issues(monitorFormSchema.safeParse(values({ auth_method: "none", clear_basic_password: true })))).toEqual([]);
    });
  });

  describe("auth_method bearer", () => {
    it("requires a token on create", () => {
      expect(issues(monitorFormSchema.safeParse(values({ auth_method: "bearer" })))).toEqual([
        ["bearer_token", "Enter a token"],
      ]);
    });

    it("accepts a typed or stored token", () => {
      expect(issues(monitorFormSchema.safeParse(values({ auth_method: "bearer", bearer_token: "t" })))).toEqual([]);
      expect(issues(monitorFormSchema.safeParse(values({ auth_method: "bearer", has_bearer_token: true })))).toEqual([]);
    });

    it("rejects clearing the stored token while bearer stays selected", () => {
      expect(
        issues(monitorFormSchema.safeParse(values({ auth_method: "bearer", has_bearer_token: true, clear_bearer_token: true }))),
      ).toEqual([["bearer_token", "Enter a token"]]);
    });
  });
});

describe("notificationFormSchema", () => {
  it("validates discord webhook URLs", () => {
    const base = newNotification("discord");
    const parse = (webhook_url: string) =>
      issues(notificationFormSchema.safeParse({ ...base, config: { ...base.config, webhook_url } }));

    expect(parse("https://discord.com/api/webhooks/123/abc")).toEqual([]);
    expect(parse("https://ptb.discordapp.com/api/webhooks/123/abc")).toEqual([]);
    expect(parse("")).toEqual([["config.webhook_url", "Enter a Discord webhook URL"]]);
    expect(parse("http://discord.com/api/webhooks/123/abc")).toEqual([
      ["config.webhook_url", "Enter an https://discord.com/api/webhooks/... URL"],
    ]);
    expect(parse("https://example.com/api/webhooks/123/abc")).toEqual([
      ["config.webhook_url", "Enter an https://discord.com/api/webhooks/... URL"],
    ]);
    expect(parse("https://discord.com/channels/123")).toEqual([
      ["config.webhook_url", "Enter an https://discord.com/api/webhooks/... URL"],
    ]);
  });

  it("validates webpush endpoint and keys", () => {
    const base = newNotification("webpush");
    expect(issues(notificationFormSchema.safeParse(base))).toEqual([
      ["config.endpoint", "Enter a push endpoint"],
      ["config.keys.p256dh", "Enter the p256dh key"],
      ["config.keys.auth", "Enter the auth key"],
    ]);
    expect(
      issues(
        notificationFormSchema.safeParse({
          ...base,
          config: { endpoint: "http://push.example/x", keys: { p256dh: "a", auth: "b" } },
        }),
      ),
    ).toEqual([["config.endpoint", "Enter an https URL"]]);
    expect(
      issues(
        notificationFormSchema.safeParse({
          ...base,
          config: { endpoint: "https://push.example/x", keys: { p256dh: "a", auth: "b" } },
        }),
      ),
    ).toEqual([]);
  });

  it("validates webhook url and custom body", () => {
    const base = newNotification("webhook");
    expect(issues(notificationFormSchema.safeParse(base))).toEqual([["config.post_url", "Enter a URL"]]);
    expect(
      issues(
        notificationFormSchema.safeParse({
          ...base,
          config: { ...base.config, post_url: "https://hooks.example.com/nocu", request_body: "custom" },
        }),
      ),
    ).toEqual([["config.custom_body", "Enter a request body"]]);
    expect(
      issues(
        notificationFormSchema.safeParse({
          ...base,
          config: {
            ...base.config,
            post_url: "http://hooks.example.com/nocu",
            request_body: "custom",
            custom_body: "{{status}}",
            additional_headers: [{ key: "", value: "x" }],
          },
        }),
      ),
    ).toEqual([["config.additional_headers.0.key", "Enter a header name"]]);
  });

  it("rejects an unknown type", () => {
    expect(notificationFormSchema.safeParse({ type: "sms", enabled: true, config: {} }).success).toBe(false);
  });
});

describe("headerRowsToRecord", () => {
  it("trims keys, keeps empty values and drops keyless rows", () => {
    expect(
      headerRowsToRecord([
        { key: " X-Probe ", value: "nocu" },
        { key: "X-Empty", value: "" },
        { key: "", value: "ignored" },
      ]),
    ).toEqual({ "X-Probe": "nocu", "X-Empty": "" });
  });
});

describe("toNotificationInput / fromNotification", () => {
  it("omits empty optional discord fields like Go's omitempty", () => {
    expect(toNotificationInput(newNotification("discord"))).toEqual({
      type: "discord",
      enabled: true,
      config: { webhook_url: "", message_format: "text", disable_url: false },
    });
  });

  it("omits custom_body unless request_body is custom and drops empty headers", () => {
    const base = newNotification("webhook");
    expect(
      toNotificationInput({ ...base, config: { ...base.config, post_url: "https://h.example", custom_body: "stale" } }),
    ).toEqual({
      type: "webhook",
      enabled: true,
      config: { post_url: "https://h.example", http_method: "POST", request_body: "json" },
    });
  });

  it("round-trips the stored channels", () => {
    for (const n of stored.notifications) {
      const input: Record<string, unknown> = { ...n };
      delete input.id;
      delete input.monitor_id;
      expect(toNotificationInput(fromNotification(n))).toEqual(input);
    }
  });
});

describe("toCreateRequest", () => {
  it("builds the Bruno create body from form values", () => {
    const body = toCreateRequest(
      values({
        retries: 1,
        cert_expiry_notification: true,
        headers: [{ key: "X-Probe", value: "nocu" }],
        notifications: stored.notifications.map(fromNotification),
      }),
    );
    expect(body).toEqual({
      type: "http",
      name: "Public website",
      url: "https://example.com/health",
      interval_seconds: 60,
      retries: 1,
      timeout_seconds: 30,
      cert_expiry_notification: true,
      domain_expiry_notification: false,
      max_redirects: 10,
      save_error_response: false,
      method: "GET",
      body_encoding: "json",
      body: "",
      headers: { "X-Probe": "nocu" },
      auth_method: "none",
      basic_username: "",
      notifications: [
        {
          type: "discord",
          enabled: true,
          config: {
            webhook_url: "https://discord.com/api/webhooks/123/abc",
            bot_display_name: "Nocu",
            message_format: "embed",
            disable_url: false,
          },
        },
        {
          type: "webhook",
          enabled: true,
          config: {
            post_url: "https://hooks.example.com/nocu",
            http_method: "POST",
            request_body: "json",
            additional_headers: { "X-Token": "secret" },
          },
        },
      ],
    });
    expect(body).not.toHaveProperty("has_basic_password");
    expect(body).not.toHaveProperty("clear_basic_password");
  });

  it("includes secrets only for the selected auth method and only when typed", () => {
    expect(toCreateRequest(values({ auth_method: "basic", basic_username: "u", basic_password: "p", bearer_token: "t" }))).toMatchObject({
      auth_method: "basic",
      basic_username: "u",
      basic_password: "p",
    });
    expect(toCreateRequest(values({ auth_method: "basic", basic_username: "u", basic_password: "p" }))).not.toHaveProperty(
      "bearer_token",
    );
    expect(toCreateRequest(values({ auth_method: "bearer", bearer_token: " t " }))).toMatchObject({ bearer_token: "t" });
    expect(toCreateRequest(values({ auth_method: "none", basic_password: "p", bearer_token: "t" }))).not.toHaveProperty(
      "basic_password",
    );
    expect(toCreateRequest(values({ auth_method: "none" }))).not.toHaveProperty("bearer_token");
  });
});

describe("fromMonitor", () => {
  it("maps a stored monitor into form values with empty secrets and has_* flags", () => {
    const form = fromMonitor({ ...stored, auth_method: "basic", basic_username: "u", has_basic_password: true });
    expect(form).toMatchObject({
      name: "Public website",
      headers: [{ key: "X-Probe", value: "nocu" }],
      auth_method: "basic",
      basic_username: "u",
      basic_password: "",
      bearer_token: "",
      has_basic_password: true,
      has_bearer_token: false,
      clear_basic_password: false,
      clear_bearer_token: false,
    });
    expect(form.notifications[0]).toEqual({
      type: "discord",
      enabled: true,
      config: {
        webhook_url: "https://discord.com/api/webhooks/123/abc",
        bot_display_name: "Nocu",
        prefix_custom_message: "",
        message_format: "embed",
        disable_url: false,
      },
    });
    expect(form.notifications[1]).toMatchObject({
      type: "webhook",
      config: { custom_body: "", additional_headers: [{ key: "X-Token", value: "secret" }] },
    });
  });

  it("treats null headers and a missing username as empty", () => {
    const form = fromMonitor({ ...stored, headers: null, basic_username: undefined });
    expect(form.headers).toEqual([]);
    expect(form.basic_username).toBe("");
    expect(monitorFormSchema.safeParse(form).success).toBe(true);
  });
});

describe("toUpdateRequest", () => {
  it("sends nothing when the form is untouched", () => {
    expect(toUpdateRequest(fromMonitor(stored), stored)).toEqual({});
  });

  it("sends only the changed scalar fields", () => {
    const form = { ...fromMonitor(stored), name: "Public website (renamed)", interval_seconds: 120, timeout_seconds: 45 };
    expect(toUpdateRequest(form, stored)).toEqual({
      name: "Public website (renamed)",
      interval_seconds: 120,
      timeout_seconds: 45,
    });
  });

  it("sends headers when a row changed", () => {
    const form = { ...fromMonitor(stored), headers: [{ key: "X-Probe", value: "nocu" }, { key: "X-Env", value: "prod" }] };
    expect(toUpdateRequest(form, stored)).toEqual({ headers: { "X-Probe": "nocu", "X-Env": "prod" } });
  });

  it("includes notifications only when the set changed", () => {
    const form = fromMonitor(stored);
    const reordered = { ...form, notifications: [...form.notifications].reverse() };
    expect(toUpdateRequest(reordered, stored)).toHaveProperty("notifications");

    const toggled = {
      ...form,
      notifications: form.notifications.map((n, i) => (i === 0 ? { ...n, enabled: false } : n)),
    };
    expect(toUpdateRequest(toggled, stored).notifications).toEqual([
      { ...toNotificationInput(form.notifications[0]), enabled: false },
      toNotificationInput(form.notifications[1]),
    ]);

    const removed = { ...form, notifications: [] };
    expect(toUpdateRequest(removed, stored)).toEqual({ notifications: [] });
  });

  it("ignores cosmetic differences in notifications (empty optionals, whitespace)", () => {
    const form = fromMonitor(stored);
    const first = form.notifications[0];
    if (first.type !== "discord") throw new Error("expected discord");
    const cosmetic = {
      ...form,
      notifications: [
        { ...first, config: { ...first.config, prefix_custom_message: "  ", webhook_url: ` ${first.config.webhook_url} ` } },
        form.notifications[1],
      ],
    };
    expect(toUpdateRequest(cosmetic, stored)).toEqual({});
  });

  describe("secrets", () => {
    const basic: Monitor = { ...stored, auth_method: "basic", basic_username: "u", has_basic_password: true };
    const bearer: Monitor = { ...stored, auth_method: "bearer", has_bearer_token: true };

    it("omits untouched secrets so the server keeps them", () => {
      expect(toUpdateRequest(fromMonitor(basic), basic)).toEqual({});
      expect(toUpdateRequest(fromMonitor(bearer), bearer)).toEqual({});
    });

    it("sends a newly typed secret", () => {
      expect(toUpdateRequest({ ...fromMonitor(basic), basic_password: "new" }, basic)).toEqual({ basic_password: "new" });
      expect(toUpdateRequest({ ...fromMonitor(bearer), bearer_token: "eyJ" }, bearer)).toEqual({ bearer_token: "eyJ" });
    });

    it('sends "" only when the user explicitly cleared a stored secret', () => {
      expect(
        toUpdateRequest({ ...fromMonitor(basic), auth_method: "none", clear_basic_password: true }, basic),
      ).toEqual({ auth_method: "none", basic_username: "", basic_password: "" });
      expect(toUpdateRequest({ ...fromMonitor(bearer), auth_method: "none", clear_bearer_token: true }, bearer)).toEqual({
        auth_method: "none",
        bearer_token: "",
      });
    });

    it("does not send a clear for a secret that was never stored", () => {
      expect(toUpdateRequest({ ...fromMonitor(stored), clear_basic_password: true, clear_bearer_token: true }, stored)).toEqual(
        {},
      );
    });

    it("switching to bearer with a new token sends auth_method and the token (Bruno update example)", () => {
      expect(toUpdateRequest({ ...fromMonitor(stored), auth_method: "bearer", bearer_token: "eyJhbGciOi..." }, stored)).toEqual({
        auth_method: "bearer",
        bearer_token: "eyJhbGciOi...",
      });
    });
  });
});
