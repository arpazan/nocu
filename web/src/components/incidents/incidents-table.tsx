"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { PlusIcon } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

import { IncidentFormDialog } from "@/components/incidents/incident-form-dialog";
import { incidentColumns } from "@/components/incidents/incidents-columns";
import { NeedsOperator } from "@/components/incidents/needs-operator";
import { DataTable, NO_ROWS } from "@/components/list/data-table";
import { EmptyState } from "@/components/list/empty-state";
import { ErrorPanel } from "@/components/list/error-panel";
import { FilterField } from "@/components/list/filter-field";
import { ListFooter } from "@/components/list/list-footer";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { INCIDENT_PAGE_SIZE, incidentsQueryOptions } from "@/lib/api/incidents";
import type { IncidentListFilters, IncidentStatus, Severity } from "@/lib/api/types";
import { canWrite } from "@/lib/auth/roles";
import { useRoles } from "@/lib/auth/use-me";
import {
  INCIDENT_STATUSES,
  SEVERITIES,
  incidentStatusLabel,
  severityLabel,
} from "@/lib/incidents/transitions";
import { cn } from "@/lib/utils";

/** Radix Select items cannot have an empty value, so "all" stands in for no filter. */
const ALL = "all";

function isStatus(value: string | null): value is IncidentStatus {
  return (INCIDENT_STATUSES as readonly string[]).includes(value ?? "");
}

function isSeverity(value: string | null): value is Severity {
  return (SEVERITIES as readonly string[]).includes(value ?? "");
}

/** Filters live in the URL so a filtered list can be shared; anything unknown is ignored. */
function readFilters(params: URLSearchParams) {
  const status = params.get("status");
  const severity = params.get("severity");
  const page = Number.parseInt(params.get("page") ?? "1", 10);
  return {
    status: isStatus(status) ? status : undefined,
    severity: isSeverity(severity) ? severity : undefined,
    page: Number.isFinite(page) && page > 0 ? page : 1,
  };
}

export function IncidentsTable() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const roles = useRoles();
  const allowed = canWrite(roles);
  const [creating, setCreating] = useState(false);

  const { status, severity, page } = readFilters(searchParams);
  const hasFilters = status !== undefined || severity !== undefined;
  const offset = (page - 1) * INCIDENT_PAGE_SIZE;

  const filters: IncidentListFilters = { limit: INCIDENT_PAGE_SIZE, offset };
  if (status) filters.status = status;
  if (severity) filters.severity = severity;

  const list = useQuery({ ...incidentsQueryOptions(filters), placeholderData: keepPreviousData });

  function navigate(changes: Record<string, string | null>) {
    const next = new URLSearchParams(searchParams.toString());
    for (const [key, value] of Object.entries(changes)) {
      if (value === null) next.delete(key);
      else next.set(key, value);
    }
    const qs = next.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  }

  // Changing a filter always goes back to the first page.
  const setStatus = (value: string) => navigate({ status: value === ALL ? null : value, page: null });
  const setSeverity = (value: string) => navigate({ severity: value === ALL ? null : value, page: null });
  const clearFilters = () => navigate({ status: null, severity: null, page: null });
  const setPage = (next: number) => navigate({ page: next <= 1 ? null : String(next) });

  const items = list.data?.items ?? NO_ROWS;
  const total = list.data?.meta.total ?? 0;
  const empty = list.isSuccess && items.length === 0;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Incidents</h1>
          <p className="text-sm text-muted-foreground">Every incident, newest first.</p>
        </div>
        <NeedsOperator allowed={allowed}>
          <Button onClick={() => setCreating(true)} disabled={!allowed}>
            <PlusIcon data-icon="inline-start" aria-hidden="true" />
            New incident
          </Button>
        </NeedsOperator>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <FilterField id="filter-status" label="Status">
          <Select value={status ?? ALL} onValueChange={setStatus}>
            <SelectTrigger id="filter-status" className={cn("w-40", status && "bg-secondary text-secondary-foreground")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>All</SelectItem>
              {INCIDENT_STATUSES.map((value) => (
                <SelectItem key={value} value={value}>
                  {incidentStatusLabel[value]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>
        <FilterField id="filter-severity" label="Severity">
          <Select value={severity ?? ALL} onValueChange={setSeverity}>
            <SelectTrigger
              id="filter-severity"
              className={cn("w-40", severity && "bg-secondary text-secondary-foreground")}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>All</SelectItem>
              {SEVERITIES.map((value) => (
                <SelectItem key={value} value={value}>
                  {severityLabel[value]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FilterField>
        {hasFilters && (
          <Button variant="ghost" onClick={clearFilters}>
            Clear
          </Button>
        )}
      </div>

      {list.isError ? (
        <ErrorPanel error={list.error} title="Could not load incidents" onRetry={() => void list.refetch()} />
      ) : empty ? (
        <EmptyState>
          {hasFilters ? "No incidents match these filters." : "No incidents yet. Create one when something needs attention."}
        </EmptyState>
      ) : (
        <div className={cn("space-y-4", list.isPlaceholderData && list.isFetching && "opacity-60 transition-opacity")}>
          <DataTable columns={incidentColumns} data={items} isPending={list.isPending} />

          <ListFooter
            offset={offset}
            shown={items.length}
            total={total}
            pageSize={INCIDENT_PAGE_SIZE}
            page={page}
            onPage={setPage}
          />
        </div>
      )}

      <IncidentFormDialog open={creating} onOpenChange={setCreating} />
    </div>
  );
}

/** Suspense fallback for the list page while the search params resolve. */
export function IncidentsTableSkeleton() {
  return (
    <div className="space-y-6" aria-busy="true" aria-label="Loading incidents">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Incidents</h1>
        <p className="text-sm text-muted-foreground">Every incident, newest first.</p>
      </div>
      <div className="flex gap-3">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-8 w-40" />
      </div>
      <DataTable columns={incidentColumns} data={NO_ROWS} isPending />
    </div>
  );
}
