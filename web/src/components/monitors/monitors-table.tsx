"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { PlusIcon } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo } from "react";

import { NeedsOperator } from "@/components/incidents/needs-operator";
import { DataTable, NO_ROWS } from "@/components/list/data-table";
import { EmptyState } from "@/components/list/empty-state";
import { ErrorPanel } from "@/components/list/error-panel";
import { FilterField } from "@/components/list/filter-field";
import { ListFooter } from "@/components/list/list-footer";
import { monitorColumns } from "@/components/monitors/monitors-columns";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { MONITOR_PAGE_SIZE, monitorsQueryOptions } from "@/lib/api/monitors";
import type { MonitorStatus } from "@/lib/api/types";
import { canDelete, canWrite } from "@/lib/auth/roles";
import { useRoles } from "@/lib/auth/use-me";
import { cn } from "@/lib/utils";

const ACTIVE_OPTIONS = ["all", "active", "paused"] as const;
const STATUS_OPTIONS = ["all", "up", "down", "pending"] as const;

type ActiveFilter = (typeof ACTIVE_OPTIONS)[number];
type StatusFilter = (typeof STATUS_OPTIONS)[number];

function pick<T extends string>(value: string | null, options: readonly T[], fallback: T): T {
  return (options as readonly string[]).includes(value ?? "") ? (value as T) : fallback;
}

/** Filters live in the URL so a page can be shared and the back button works. */
function useListFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const active = pick<ActiveFilter>(searchParams.get("active"), ACTIVE_OPTIONS, "all");
  const status = pick<StatusFilter>(searchParams.get("status"), STATUS_OPTIONS, "all");
  const page = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);

  const update = useCallback(
    (next: { active?: ActiveFilter; status?: StatusFilter; page?: number }) => {
      const params = new URLSearchParams(searchParams.toString());
      const set = (key: string, value: string | undefined, defaultValue: string) => {
        if (value === undefined) return;
        if (value === defaultValue) params.delete(key);
        else params.set(key, value);
      };
      set("active", next.active, "all");
      set("status", next.status, "all");
      // Changing a filter restarts from the first page.
      const page = next.page ?? (next.active !== undefined || next.status !== undefined ? 1 : undefined);
      set("page", page === undefined ? undefined : String(page), "1");
      const query = params.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams],
  );

  const clear = useCallback(() => router.replace(pathname, { scroll: false }), [pathname, router]);

  return { active, status, page, update, clear, isFiltered: active !== "all" || status !== "all" };
}

function NewMonitorButton({ writable }: { writable: boolean }) {
  if (writable) {
    return (
      <Button asChild>
        <Link href="/monitors/new">
          <PlusIcon data-icon="inline-start" aria-hidden="true" />
          New monitor
        </Link>
      </Button>
    );
  }
  return (
    <NeedsOperator allowed={false}>
      <Button disabled aria-disabled="true">
        <PlusIcon data-icon="inline-start" aria-hidden="true" />
        New monitor
      </Button>
    </NeedsOperator>
  );
}

export function MonitorsTable() {
  const filters = useListFilters();
  const roles = useRoles();
  const writable = canWrite(roles);
  const deletable = canDelete(roles);
  const columns = useMemo(() => monitorColumns({ writable, deletable }), [writable, deletable]);

  const offset = (filters.page - 1) * MONITOR_PAGE_SIZE;
  const query = useQuery({
    ...monitorsQueryOptions({
      active: filters.active === "all" ? undefined : filters.active === "active",
      status: filters.status === "all" ? undefined : (filters.status as MonitorStatus),
      limit: MONITOR_PAGE_SIZE,
      offset,
    }),
    placeholderData: keepPreviousData,
  });

  const items = query.data?.items ?? NO_ROWS;
  const total = query.data?.meta.total ?? 0;
  const empty = query.isSuccess && items.length === 0;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Monitors</h1>
          <p className="text-sm text-muted-foreground">Every URL being checked, with its recent heartbeats.</p>
        </div>
        <NewMonitorButton writable={writable} />
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <FilterField id="filter-active" label="Checks">
          <Select value={filters.active} onValueChange={(value) => filters.update({ active: value as ActiveFilter })}>
            <SelectTrigger
              id="filter-active"
              className={cn("w-36", filters.active !== "all" && "bg-secondary text-secondary-foreground")}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="active">Running</SelectItem>
              <SelectItem value="paused">Paused</SelectItem>
            </SelectContent>
          </Select>
        </FilterField>
        <FilterField id="filter-status" label="Status">
          <Select value={filters.status} onValueChange={(value) => filters.update({ status: value as StatusFilter })}>
            <SelectTrigger
              id="filter-status"
              className={cn("w-44", filters.status !== "all" && "bg-secondary text-secondary-foreground")}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="up">Up</SelectItem>
              <SelectItem value="down">Down</SelectItem>
              <SelectItem value="pending">Waiting for first check</SelectItem>
            </SelectContent>
          </Select>
        </FilterField>
        {filters.isFiltered ? (
          <Button variant="ghost" onClick={filters.clear}>
            Clear
          </Button>
        ) : null}
      </div>

      {query.isError ? (
        <ErrorPanel error={query.error} title="Could not load monitors" onRetry={() => void query.refetch()} />
      ) : empty ? (
        <EmptyState>
          {filters.isFiltered
            ? "No monitors match these filters."
            : "No monitors yet. Add your first URL to start checking it."}
        </EmptyState>
      ) : (
        <div className={cn("space-y-4", query.isPlaceholderData && query.isFetching && "opacity-60 transition-opacity")}>
          <DataTable columns={columns} data={items} isPending={query.isPending} />

          <ListFooter
            offset={offset}
            shown={items.length}
            total={total}
            pageSize={MONITOR_PAGE_SIZE}
            page={filters.page}
            onPage={(page) => filters.update({ page })}
          />
        </div>
      )}
    </div>
  );
}
