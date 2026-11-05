import { queryOptions } from "@tanstack/react-query";

import { authedCoreApi } from "@/lib/api/authed";
import type {
  CreateMonitorRequest,
  Heartbeat,
  HeartbeatListFilters,
  Monitor,
  MonitorListFilters,
  UpdateMonitorRequest,
} from "@/lib/api/types";

/** Default page size for monitor and heartbeat lists. */
export const MONITOR_PAGE_SIZE = 20;

const BASE = "/api/v1/monitors";

/** Path for one monitor. */
export function monitorPath(id: string): string {
  return `${BASE}/${encodeURIComponent(id)}`;
}

/**
 * Query keys, nested so a parent key invalidates everything below it:
 * `monitorKeys.all` clears lists, details and heartbeats at once.
 */
export const monitorKeys = {
  all: ["monitors"] as const,
  lists: () => [...monitorKeys.all, "list"] as const,
  list: (filters: MonitorListFilters) => [...monitorKeys.lists(), filters] as const,
  details: () => [...monitorKeys.all, "detail"] as const,
  detail: (id: string) => [...monitorKeys.details(), id] as const,
  heartbeats: (id: string, filters: HeartbeatListFilters) =>
    [...monitorKeys.detail(id), "heartbeats", filters] as const,
};

/** Fills in the default page size so the key and the request agree. */
function withPageDefaults<T extends { limit?: number; offset?: number }>(filters: T): T {
  return { ...filters, limit: filters.limit ?? MONITOR_PAGE_SIZE, offset: filters.offset ?? 0 };
}

/** GET /api/v1/monitors, newest first. */
export function monitorsQueryOptions(filters: MonitorListFilters = {}) {
  const query = withPageDefaults(filters);
  return queryOptions({
    queryKey: monitorKeys.list(query),
    queryFn: () => authedCoreApi.list<Monitor>(BASE, { query }),
  });
}

/** GET /api/v1/monitors/:id with its notifications. */
export function monitorQueryOptions(id: string) {
  return queryOptions({
    queryKey: monitorKeys.detail(id),
    queryFn: () => authedCoreApi.get<Monitor>(monitorPath(id)),
  });
}

/** GET /api/v1/monitors/:id/heartbeats, newest first. */
export function heartbeatsQueryOptions(id: string, filters: HeartbeatListFilters = {}) {
  const query = withPageDefaults(filters);
  return queryOptions({
    queryKey: monitorKeys.heartbeats(id, query),
    queryFn: () => authedCoreApi.list<Heartbeat>(`${monitorPath(id)}/heartbeats`, { query }),
  });
}

/** POST /api/v1/monitors. Requires operator or admin. */
export function createMonitor(body: CreateMonitorRequest): Promise<Monitor> {
  return authedCoreApi.post<Monitor>(BASE, body);
}

/** PATCH /api/v1/monitors/:id. Only present fields change. */
export function updateMonitor(id: string, body: UpdateMonitorRequest): Promise<Monitor> {
  return authedCoreApi.patch<Monitor>(monitorPath(id), body);
}

/** POST /api/v1/monitors/:id/pause. Sets active: false. */
export function pauseMonitor(id: string): Promise<Monitor> {
  return authedCoreApi.post<Monitor>(`${monitorPath(id)}/pause`);
}

/** POST /api/v1/monitors/:id/resume. Sets active: true. */
export function resumeMonitor(id: string): Promise<Monitor> {
  return authedCoreApi.post<Monitor>(`${monitorPath(id)}/resume`);
}

/** DELETE /api/v1/monitors/:id. Soft delete, 204 on success. Requires admin. */
export function deleteMonitor(id: string): Promise<void> {
  return authedCoreApi.delete(monitorPath(id));
}
