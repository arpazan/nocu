import { queryOptions } from "@tanstack/react-query";

import { authedCoreApi } from "@/lib/api/authed";
import type {
  CreateIncidentRequest,
  Incident,
  IncidentListFilters,
  UpdateIncidentRequest,
} from "@/lib/api/types";

/** Default page size for incident lists. */
export const INCIDENT_PAGE_SIZE = 20;

const BASE = "/api/v1/incidents";

/** Path for one incident. */
export function incidentPath(id: string): string {
  return `${BASE}/${encodeURIComponent(id)}`;
}

/** Query keys, nested so `incidentKeys.all` invalidates lists and details. */
export const incidentKeys = {
  all: ["incidents"] as const,
  lists: () => [...incidentKeys.all, "list"] as const,
  list: (filters: IncidentListFilters) => [...incidentKeys.lists(), filters] as const,
  details: () => [...incidentKeys.all, "detail"] as const,
  detail: (id: string) => [...incidentKeys.details(), id] as const,
};

/** GET /api/v1/incidents, newest first. */
export function incidentsQueryOptions(filters: IncidentListFilters = {}) {
  const query: IncidentListFilters = {
    ...filters,
    limit: filters.limit ?? INCIDENT_PAGE_SIZE,
    offset: filters.offset ?? 0,
  };
  return queryOptions({
    queryKey: incidentKeys.list(query),
    queryFn: () => authedCoreApi.list<Incident>(BASE, { query }),
  });
}

/** GET /api/v1/incidents/:id. */
export function incidentQueryOptions(id: string) {
  return queryOptions({
    queryKey: incidentKeys.detail(id),
    queryFn: () => authedCoreApi.get<Incident>(incidentPath(id)),
  });
}

/** POST /api/v1/incidents. Starts as `open`. Requires operator or admin. */
export function createIncident(body: CreateIncidentRequest): Promise<Incident> {
  return authedCoreApi.post<Incident>(BASE, body);
}

/**
 * PATCH /api/v1/incidents/:id. Only present fields change; an illegal status
 * transition is rejected with a 409 `conflict` ApiError.
 */
export function updateIncident(id: string, body: UpdateIncidentRequest): Promise<Incident> {
  return authedCoreApi.patch<Incident>(incidentPath(id), body);
}

/** DELETE /api/v1/incidents/:id. Soft delete, 204 on success. Requires admin. */
export function deleteIncident(id: string): Promise<void> {
  return authedCoreApi.delete(incidentPath(id));
}
