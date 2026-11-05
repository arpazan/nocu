import type { AuthMethod, BodyEncoding, MonitorStatus, NotificationType } from "@/lib/api/types";

export const monitorStatusLabel: Record<MonitorStatus, string> = {
  pending: "Waiting for first check",
  up: "Up",
  down: "Down",
};

export const notificationTypeLabel: Record<NotificationType, string> = {
  discord: "Discord",
  webpush: "Browser push",
  webhook: "Webhook",
};

export const authMethodLabel: Record<AuthMethod, string> = {
  none: "None",
  basic: "Basic auth",
  bearer: "Bearer token",
};

export const bodyEncodingLabel: Record<BodyEncoding, string> = {
  json: "JSON",
  form: "Form (urlencoded)",
  xml: "XML",
};
