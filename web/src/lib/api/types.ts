/**
 * Wire types mirroring the Go services. Field names are snake_case because
 * that is what the GORM models and httpx envelope emit; keep them in sync with
 * shared/httpx/response.go, shared/httpx/errors.go and shared/auth/jwt.go.
 */

/** Successful response: shared/httpx/response.go Envelope. */
export interface Envelope<T> {
  data: T;
  meta?: Meta;
}

/** Pagination details attached to list responses. */
export interface Meta {
  total: number;
  limit: number;
  offset: number;
}

/** Error response: shared/httpx/errors.go errorBody. */
export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

/** identity /me and login user payload. */
export interface User {
  id: string;
  email: string;
  name: string;
  is_active: boolean;
  roles: string[];
  created_at: string;
}

/** identity login and refresh payload: shared/auth/jwt.go TokenPair. */
export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: "Bearer";
  expires_in: number;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest extends LoginRequest {
  name: string;
}

/** /healthz on every HTTP service. */
export interface Health {
  status: string;
}

// ---------------------------------------------------------------------------
// Monitors: shared/model/monitor.go, shared/model/monitor_notification.go
// ---------------------------------------------------------------------------

/** shared/model/monitor.go MonitorStatus. */
export type MonitorStatus = "pending" | "up" | "down";

/** shared/model/monitor.go HTTPMethod. */
export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS";

/** shared/model/monitor.go BodyEncoding. */
export type BodyEncoding = "json" | "form" | "xml";

/** shared/model/monitor.go AuthMethod. */
export type AuthMethod = "none" | "basic" | "bearer";

/** shared/model/monitor_notification.go NotificationType. */
export type NotificationType = "discord" | "webpush" | "webhook";

/** shared/model/monitor_notification.go DiscordConfig. */
export interface DiscordConfig {
  webhook_url: string;
  /** omitempty */
  bot_display_name?: string;
  /** omitempty */
  prefix_custom_message?: string;
  message_format: "text" | "embed";
  disable_url: boolean;
}

/** shared/model/monitor_notification.go WebPushConfig (PushSubscription.toJSON()). */
export interface WebPushConfig {
  endpoint: string;
  keys: {
    p256dh: string;
    auth: string;
  };
}

/** shared/model/monitor_notification.go WebhookConfig. */
export interface WebhookConfig {
  post_url: string;
  http_method: "POST" | "PUT" | "PATCH";
  request_body: "json" | "form" | "custom";
  /** omitempty; required when request_body is "custom". */
  custom_body?: string;
  /** omitempty */
  additional_headers?: Record<string, string>;
}

/** Config payload keyed by notification type. */
export interface NotificationConfigMap {
  discord: DiscordConfig;
  webpush: WebPushConfig;
  webhook: WebhookConfig;
}

/**
 * shared/model/monitor_notification.go MonitorNotification as a discriminated
 * union on `type`, so `config` narrows with it.
 */
export type MonitorNotification = {
  [K in NotificationType]: {
    id: string;
    monitor_id: string;
    type: K;
    enabled: boolean;
    config: NotificationConfigMap[K];
    created_at?: string;
    updated_at?: string;
  };
}[NotificationType];

/** Notification as sent in create/update bodies (ids are server-assigned). */
export type MonitorNotificationInput = {
  [K in NotificationType]: {
    type: K;
    enabled: boolean;
    config: NotificationConfigMap[K];
  };
}[NotificationType];

/**
 * shared/model/monitor.go Monitor. Secrets are never serialised; MarshalJSON
 * adds has_basic_password / has_bearer_token instead. Nullable timestamps are
 * `omitempty` pointers, so they arrive as null, absent, or an RFC 3339 string.
 */
export interface Monitor {
  id: string;
  type: "http";
  name: string;
  url: string;
  interval_seconds: number;
  retries: number;
  timeout_seconds: number;
  active: boolean;
  /** omitempty */
  owner_id?: string | null;
  cert_expiry_notification: boolean;
  domain_expiry_notification: boolean;
  max_redirects: number;
  save_error_response: boolean;
  /** Open a critical incident automatically when the monitor flips to down. */
  auto_create_incident: boolean;
  method: HttpMethod;
  body_encoding: BodyEncoding;
  body: string;
  /** JSON object of string values; a never-set column may serialise as null. */
  headers: Record<string, string> | null;
  auth_method: AuthMethod;
  /** omitempty */
  basic_username?: string;
  has_basic_password: boolean;
  has_bearer_token: boolean;
  last_status: MonitorStatus;
  /** omitempty */
  last_checked_at?: string | null;
  /** omitempty */
  next_check_at?: string | null;
  failure_count: number;
  /** omitempty */
  cert_expires_at?: string | null;
  /** omitempty */
  domain_expires_at?: string | null;
  notifications: MonitorNotification[];
  created_at: string;
  updated_at: string;
}

/** shared/model/monitor.go MonitorHeartbeat. */
export interface Heartbeat {
  id: string;
  monitor_id: string;
  status: MonitorStatus;
  latency_ms: number;
  http_status: number;
  message: string;
  /** omitempty; only present when the monitor has save_error_response on. */
  error_body?: string;
  /** True when this probe flipped the monitor status. */
  important: boolean;
  checked_at: string;
  created_at: string;
  updated_at: string;
}

/** POST /api/v1/monitors body. Only name and url are required; see bruno/core-api/monitors/folder.bru for defaults. */
export interface CreateMonitorRequest {
  type?: "http";
  name: string;
  url: string;
  interval_seconds?: number;
  retries?: number;
  timeout_seconds?: number;
  cert_expiry_notification?: boolean;
  domain_expiry_notification?: boolean;
  max_redirects?: number;
  save_error_response?: boolean;
  auto_create_incident?: boolean;
  method?: HttpMethod;
  body_encoding?: BodyEncoding;
  body?: string;
  headers?: Record<string, string>;
  auth_method?: AuthMethod;
  basic_username?: string;
  /** Write-only. */
  basic_password?: string;
  /** Write-only. */
  bearer_token?: string;
  notifications?: MonitorNotificationInput[];
}

/**
 * PATCH /api/v1/monitors/:id body. Only present fields change; secrets sent
 * as "" are cleared; `notifications`, when present, replaces the whole set.
 */
export type UpdateMonitorRequest = Partial<CreateMonitorRequest>;

/** GET /api/v1/monitors query. */
export type MonitorListFilters = {
  active?: boolean;
  status?: MonitorStatus;
  limit?: number;
  offset?: number;
};

/** GET /api/v1/monitors/:id/heartbeats query. */
export type HeartbeatListFilters = {
  important?: boolean;
  limit?: number;
  offset?: number;
};

// ---------------------------------------------------------------------------
// Incidents: shared/model/incident.go
// ---------------------------------------------------------------------------

/** shared/model/incident.go Severity. */
export type Severity = "critical" | "high" | "medium" | "low";

/** shared/model/incident.go IncidentStatus. */
export type IncidentStatus = "open" | "acknowledged" | "resolved" | "closed";

/** shared/model/incident.go Incident. Nullable timestamps are omitempty pointers. */
export interface Incident {
  id: string;
  title: string;
  description: string;
  severity: Severity;
  status: IncidentStatus;
  source: string;
  /** omitempty */
  reporter_id?: string | null;
  /** omitempty; set when the scheduler opened the incident for a monitor. */
  monitor_id?: string | null;
  /** omitempty */
  acknowledged_at?: string | null;
  /** omitempty */
  resolved_at?: string | null;
  /** omitempty */
  closed_at?: string | null;
  created_at: string;
  updated_at: string;
}

/** POST /api/v1/incidents body. */
export interface CreateIncidentRequest {
  title: string;
  description?: string;
  severity: Severity;
  source?: string;
}

/** PATCH /api/v1/incidents/:id body. */
export interface UpdateIncidentRequest {
  title?: string;
  description?: string;
  severity?: Severity;
  status?: IncidentStatus;
}

/** GET /api/v1/incidents query. */
export type IncidentListFilters = {
  status?: IncidentStatus;
  severity?: Severity;
  monitor_id?: string;
  limit?: number;
  offset?: number;
};
