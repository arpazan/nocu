import { z } from "zod";

import type {
  CreateMonitorRequest,
  Monitor,
  MonitorNotification,
  MonitorNotificationInput,
  NotificationType,
  UpdateMonitorRequest,
} from "@/lib/api/types";

/**
 * Form-side validation for the monitor editor. Rules mirror
 * bruno/core-api/monitors/folder.bru and the Validate methods in
 * shared/model/monitor_notification.go. Values are shaped for
 * react-hook-form: every string input is a required string (empty rather
 * than undefined) and header maps are edited as key/value rows.
 */

export const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"] as const;
export const BODY_ENCODINGS = ["json", "form", "xml"] as const;
export const AUTH_METHODS = ["none", "basic", "bearer"] as const;
export const NOTIFICATION_TYPES = ["discord", "webpush", "webhook"] as const;

// ---------------------------------------------------------------------------
// Building blocks
// ---------------------------------------------------------------------------

function parseUrl(value: string): URL | null {
  try {
    return new URL(value.trim());
  } catch {
    return null;
  }
}

function isHttpUrl(value: string): boolean {
  const url = parseUrl(value);
  return url !== null && (url.protocol === "http:" || url.protocol === "https:") && url.host !== "";
}

function isHttpsUrl(value: string): boolean {
  const url = parseUrl(value);
  return url !== null && url.protocol === "https:" && url.host !== "";
}

function isDiscordWebhookUrl(value: string): boolean {
  const url = parseUrl(value);
  if (!url || url.protocol !== "https:") return false;
  const host = url.hostname.toLowerCase();
  const discordHost = ["discord.com", "discordapp.com"].some((d) => host === d || host.endsWith(`.${d}`));
  return discordHost && url.pathname.startsWith("/api/webhooks/");
}

const wholeNumber = (message: string) => z.number({ error: message }).int("Enter a whole number");

/** One editable header row. Rows with an empty key and value are ignored. */
export const headerRowSchema = z
  .object({
    key: z.string(),
    value: z.string(),
  })
  .superRefine((row, ctx) => {
    if (row.key.trim() === "" && row.value.trim() !== "") {
      ctx.addIssue({ code: "custom", path: ["key"], message: "Enter a header name" });
    }
  });

export type HeaderRow = z.infer<typeof headerRowSchema>;

export const headerRowsSchema = z.array(headerRowSchema);

// ---------------------------------------------------------------------------
// Notification channels
// ---------------------------------------------------------------------------

export const discordConfigSchema = z.object({
  webhook_url: z
    .string()
    .trim()
    .min(1, { error: "Enter a Discord webhook URL", abort: true })
    .refine(isDiscordWebhookUrl, "Enter an https://discord.com/api/webhooks/... URL"),
  bot_display_name: z.string(),
  prefix_custom_message: z.string(),
  message_format: z.enum(["text", "embed"], { error: "Choose a message format" }),
  disable_url: z.boolean(),
});

export const webPushConfigSchema = z.object({
  endpoint: z.string().trim().min(1, { error: "Enter a push endpoint", abort: true }).refine(isHttpsUrl, "Enter an https URL"),
  keys: z.object({
    p256dh: z.string().trim().min(1, "Enter the p256dh key"),
    auth: z.string().trim().min(1, "Enter the auth key"),
  }),
});

export const webhookConfigSchema = z
  .object({
    post_url: z.string().trim().min(1, { error: "Enter a URL", abort: true }).refine(isHttpUrl, "Enter an http or https URL"),
    http_method: z.enum(["POST", "PUT", "PATCH"], { error: "Choose a method" }),
    request_body: z.enum(["json", "form", "custom"], { error: "Choose a body format" }),
    custom_body: z.string(),
    additional_headers: headerRowsSchema,
  })
  .superRefine((config, ctx) => {
    if (config.request_body === "custom" && config.custom_body.trim() === "") {
      ctx.addIssue({ code: "custom", path: ["custom_body"], message: "Enter a request body" });
    }
  });

export const notificationFormSchema = z.discriminatedUnion("type", [
  z.object({ type: z.literal("discord"), enabled: z.boolean(), config: discordConfigSchema }),
  z.object({ type: z.literal("webpush"), enabled: z.boolean(), config: webPushConfigSchema }),
  z.object({ type: z.literal("webhook"), enabled: z.boolean(), config: webhookConfigSchema }),
]);

export type MonitorNotificationFormValues = z.infer<typeof notificationFormSchema>;

// ---------------------------------------------------------------------------
// Monitor form
// ---------------------------------------------------------------------------

export const monitorFormSchema = z
  .object({
    name: z.string().trim().min(1, "Enter a name").max(255, "Name must be 255 characters or fewer"),
    url: z.string().trim().min(1, { error: "Enter a URL", abort: true }).refine(isHttpUrl, "Enter an http or https URL"),
    interval_seconds: wholeNumber("Enter an interval").min(20, "At least 20 seconds"),
    retries: wholeNumber("Enter a number of retries").min(0, "Cannot be negative"),
    timeout_seconds: wholeNumber("Enter a timeout").min(1, "At least 1 second"),
    cert_expiry_notification: z.boolean(),
    domain_expiry_notification: z.boolean(),
    save_error_response: z.boolean(),
    max_redirects: wholeNumber("Enter a redirect limit").min(0, "Cannot be negative"),
    method: z.enum(HTTP_METHODS, { error: "Choose a method" }),
    body_encoding: z.enum(BODY_ENCODINGS, { error: "Choose an encoding" }),
    body: z.string(),
    headers: headerRowsSchema,
    auth_method: z.enum(AUTH_METHODS, { error: "Choose an auth method" }),
    basic_username: z.string(),
    /** Write-only; empty means "unchanged" in edit mode. */
    basic_password: z.string(),
    /** Write-only; empty means "unchanged" in edit mode. */
    bearer_token: z.string(),
    /** Edit mode: the server already holds a password (Monitor.has_basic_password). */
    has_basic_password: z.boolean().optional(),
    /** Edit mode: the server already holds a token (Monitor.has_bearer_token). */
    has_bearer_token: z.boolean().optional(),
    /** Edit mode: the user asked to clear the stored password. */
    clear_basic_password: z.boolean().optional(),
    /** Edit mode: the user asked to clear the stored token. */
    clear_bearer_token: z.boolean().optional(),
    notifications: z.array(notificationFormSchema),
  })
  .superRefine((values, ctx) => {
    if (
      Number.isFinite(values.timeout_seconds) &&
      Number.isFinite(values.interval_seconds) &&
      values.timeout_seconds >= values.interval_seconds
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["timeout_seconds"],
        message: "Timeout must be shorter than the interval",
      });
    }

    if (values.auth_method === "basic") {
      if (values.basic_username.trim() === "") {
        ctx.addIssue({ code: "custom", path: ["basic_username"], message: "Enter a username" });
      }
      const stored = values.has_basic_password === true && values.clear_basic_password !== true;
      if (values.basic_password === "" && !stored) {
        ctx.addIssue({ code: "custom", path: ["basic_password"], message: "Enter a password" });
      }
    }

    if (values.auth_method === "bearer") {
      const stored = values.has_bearer_token === true && values.clear_bearer_token !== true;
      if (values.bearer_token.trim() === "" && !stored) {
        ctx.addIssue({ code: "custom", path: ["bearer_token"], message: "Enter a token" });
      }
    }
  });

export type MonitorFormValues = z.infer<typeof monitorFormSchema>;

/** Defaults from bruno/core-api/monitors/folder.bru. */
export const monitorFormDefaults: MonitorFormValues = {
  name: "",
  url: "",
  interval_seconds: 60,
  retries: 0,
  timeout_seconds: 30,
  cert_expiry_notification: false,
  domain_expiry_notification: false,
  save_error_response: false,
  max_redirects: 10,
  method: "GET",
  body_encoding: "json",
  body: "",
  headers: [],
  auth_method: "none",
  basic_username: "",
  basic_password: "",
  bearer_token: "",
  has_basic_password: false,
  has_bearer_token: false,
  clear_basic_password: false,
  clear_bearer_token: false,
  notifications: [],
};

/** The form variant for one notification type. */
export type NotificationFormValuesOf<T extends NotificationType> = Extract<MonitorNotificationFormValues, { type: T }>;

/** A blank, enabled channel of the given type for an "add notification" action. */
export function newNotification<T extends NotificationType>(type: T): NotificationFormValuesOf<T> {
  return blankNotification(type) as NotificationFormValuesOf<T>;
}

function blankNotification(type: NotificationType): MonitorNotificationFormValues {
  switch (type) {
    case "discord":
      return {
        type,
        enabled: true,
        config: {
          webhook_url: "",
          bot_display_name: "",
          prefix_custom_message: "",
          message_format: "text",
          disable_url: false,
        },
      };
    case "webpush":
      return { type, enabled: true, config: { endpoint: "", keys: { p256dh: "", auth: "" } } };
    case "webhook":
      return {
        type,
        enabled: true,
        config: { post_url: "", http_method: "POST", request_body: "json", custom_body: "", additional_headers: [] },
      };
  }
}

// ---------------------------------------------------------------------------
// Converters: form values <-> wire types
// ---------------------------------------------------------------------------

/** Header rows -> object. Rows with a blank key are dropped; keys are trimmed. */
export function headerRowsToRecord(rows: HeaderRow[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const row of rows) {
    const key = row.key.trim();
    if (key !== "") out[key] = row.value;
  }
  return out;
}

/** Object -> header rows, in key order. */
export function recordToHeaderRows(record: Record<string, string> | null | undefined): HeaderRow[] {
  return Object.entries(record ?? {}).map(([key, value]) => ({ key, value }));
}

/** Form channel -> request channel, omitting empty optionals the way Go's omitempty does. */
export function toNotificationInput(n: MonitorNotificationFormValues): MonitorNotificationInput {
  switch (n.type) {
    case "discord": {
      const c = n.config;
      return {
        type: "discord",
        enabled: n.enabled,
        config: {
          webhook_url: c.webhook_url.trim(),
          ...(c.bot_display_name.trim() ? { bot_display_name: c.bot_display_name.trim() } : {}),
          ...(c.prefix_custom_message.trim() ? { prefix_custom_message: c.prefix_custom_message.trim() } : {}),
          message_format: c.message_format,
          disable_url: c.disable_url,
        },
      };
    }
    case "webpush": {
      const c = n.config;
      return {
        type: "webpush",
        enabled: n.enabled,
        config: {
          endpoint: c.endpoint.trim(),
          keys: { p256dh: c.keys.p256dh.trim(), auth: c.keys.auth.trim() },
        },
      };
    }
    case "webhook": {
      const c = n.config;
      const headers = headerRowsToRecord(c.additional_headers);
      return {
        type: "webhook",
        enabled: n.enabled,
        config: {
          post_url: c.post_url.trim(),
          http_method: c.http_method,
          request_body: c.request_body,
          ...(c.request_body === "custom" && c.custom_body !== "" ? { custom_body: c.custom_body } : {}),
          ...(Object.keys(headers).length > 0 ? { additional_headers: headers } : {}),
        },
      };
    }
  }
}

/** Stored channel -> form channel, filling optional config fields with empty strings. */
export function fromNotification(n: MonitorNotification | MonitorNotificationInput): MonitorNotificationFormValues {
  switch (n.type) {
    case "discord":
      return {
        type: "discord",
        enabled: n.enabled,
        config: {
          webhook_url: n.config.webhook_url,
          bot_display_name: n.config.bot_display_name ?? "",
          prefix_custom_message: n.config.prefix_custom_message ?? "",
          message_format: n.config.message_format,
          disable_url: n.config.disable_url,
        },
      };
    case "webpush":
      return {
        type: "webpush",
        enabled: n.enabled,
        config: { endpoint: n.config.endpoint, keys: { p256dh: n.config.keys.p256dh, auth: n.config.keys.auth } },
      };
    case "webhook":
      return {
        type: "webhook",
        enabled: n.enabled,
        config: {
          post_url: n.config.post_url,
          http_method: n.config.http_method,
          request_body: n.config.request_body,
          custom_body: n.config.custom_body ?? "",
          additional_headers: recordToHeaderRows(n.config.additional_headers),
        },
      };
  }
}

/** Stored channel -> request channel (drops server-assigned ids and timestamps). */
function stripNotification(n: MonitorNotification): MonitorNotificationInput {
  return toNotificationInput(fromNotification(n));
}

/** Every non-secret, non-flag field as it would be sent on create. */
function baseRequest(values: MonitorFormValues): Omit<CreateMonitorRequest, "basic_password" | "bearer_token"> {
  return {
    type: "http",
    name: values.name.trim(),
    url: values.url.trim(),
    interval_seconds: values.interval_seconds,
    retries: values.retries,
    timeout_seconds: values.timeout_seconds,
    cert_expiry_notification: values.cert_expiry_notification,
    domain_expiry_notification: values.domain_expiry_notification,
    max_redirects: values.max_redirects,
    save_error_response: values.save_error_response,
    method: values.method,
    body_encoding: values.body_encoding,
    body: values.body,
    headers: headerRowsToRecord(values.headers),
    auth_method: values.auth_method,
    basic_username: values.auth_method === "basic" ? values.basic_username.trim() : "",
    notifications: values.notifications.map(toNotificationInput),
  };
}

/** Form values -> POST body. Secrets are sent only when non-empty; the has_ and clear_ flags are dropped. */
export function toCreateRequest(values: MonitorFormValues): CreateMonitorRequest {
  const body: CreateMonitorRequest = baseRequest(values);
  if (values.auth_method === "basic" && values.basic_password !== "") body.basic_password = values.basic_password;
  if (values.auth_method === "bearer" && values.bearer_token.trim() !== "") body.bearer_token = values.bearer_token.trim();
  return body;
}

/** Structural equality for plain JSON values (key order independent). */
function deepEqual(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) return true;
  if (typeof a !== "object" || typeof b !== "object" || a === null || b === null) return false;
  if (Array.isArray(a) !== Array.isArray(b)) return false;
  if (Array.isArray(a) && Array.isArray(b)) {
    return a.length === b.length && a.every((item, i) => deepEqual(item, b[i]));
  }
  const left = a as Record<string, unknown>;
  const right = b as Record<string, unknown>;
  const keys = new Set([...Object.keys(left), ...Object.keys(right)]);
  for (const key of keys) {
    if (!deepEqual(left[key], right[key])) return false;
  }
  return true;
}

/**
 * Form values -> PATCH body containing only what changed against `original`.
 *
 * Secrets: a typed value is sent; an untouched field is omitted (server keeps
 * the stored value); `""` is sent only when the matching clear_* flag is set.
 * `notifications` is included only when the set differs, because the server
 * replaces all channels whenever the key is present.
 */
export function toUpdateRequest(values: MonitorFormValues, original: Monitor): UpdateMonitorRequest {
  const next = baseRequest(values);
  const patch: UpdateMonitorRequest = {};

  const current: Omit<CreateMonitorRequest, "basic_password" | "bearer_token" | "type"> = {
    name: original.name,
    url: original.url,
    interval_seconds: original.interval_seconds,
    retries: original.retries,
    timeout_seconds: original.timeout_seconds,
    cert_expiry_notification: original.cert_expiry_notification,
    domain_expiry_notification: original.domain_expiry_notification,
    max_redirects: original.max_redirects,
    save_error_response: original.save_error_response,
    method: original.method,
    body_encoding: original.body_encoding,
    body: original.body,
    headers: original.headers ?? {},
    auth_method: original.auth_method,
    basic_username: original.basic_username ?? "",
    notifications: original.notifications.map(stripNotification),
  };

  for (const key of Object.keys(current) as (keyof typeof current)[]) {
    if (!deepEqual(next[key], current[key])) {
      // Keys match 1:1 between the two objects; TS cannot see that across a union of keys.
      (patch as Record<string, unknown>)[key] = next[key];
    }
  }

  if (values.basic_password !== "") {
    patch.basic_password = values.basic_password;
  } else if (values.clear_basic_password === true && original.has_basic_password) {
    patch.basic_password = "";
  }

  if (values.bearer_token.trim() !== "") {
    patch.bearer_token = values.bearer_token.trim();
  } else if (values.clear_bearer_token === true && original.has_bearer_token) {
    patch.bearer_token = "";
  }

  return patch;
}

/** Stored monitor -> form values for edit mode. Secrets start empty; has_* flags carry what is stored. */
export function fromMonitor(m: Monitor): MonitorFormValues {
  return {
    name: m.name,
    url: m.url,
    interval_seconds: m.interval_seconds,
    retries: m.retries,
    timeout_seconds: m.timeout_seconds,
    cert_expiry_notification: m.cert_expiry_notification,
    domain_expiry_notification: m.domain_expiry_notification,
    save_error_response: m.save_error_response,
    max_redirects: m.max_redirects,
    method: m.method,
    body_encoding: m.body_encoding,
    body: m.body,
    headers: recordToHeaderRows(m.headers),
    auth_method: m.auth_method,
    basic_username: m.basic_username ?? "",
    basic_password: "",
    bearer_token: "",
    has_basic_password: m.has_basic_password,
    has_bearer_token: m.has_bearer_token,
    clear_basic_password: false,
    clear_bearer_token: false,
    notifications: m.notifications.map(fromNotification),
  };
}
