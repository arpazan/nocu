"use client";

import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon, ExternalLinkIcon } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { type ReactNode, useCallback, useState } from "react";

import { createListColumnHelper, DataTable, NO_ROWS } from "@/components/list/data-table";
import { HeartbeatStrip } from "@/components/monitors/heartbeat-strip";
import { DeleteMonitorDialog, useDeleteMonitor, usePauseResume } from "@/components/monitors/monitor-mutations";
import { ErrorPanel, NotFoundPanel } from "@/components/monitors/monitor-panels";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { ApiError } from "@/lib/api/client";
import { heartbeatsQueryOptions, MONITOR_PAGE_SIZE, monitorQueryOptions } from "@/lib/api/monitors";
import type { Heartbeat, Monitor, MonitorStatus } from "@/lib/api/types";
import { canDelete, canWrite } from "@/lib/auth/roles";
import { useRoles } from "@/lib/auth/use-me";
import { formatDateTime, formatInterval, formatLatency, formatRelative } from "@/lib/format";
import { authMethodLabel, monitorStatusLabel, notificationTypeLabel } from "@/lib/monitors/labels";

const STRIP_COUNT = 60;

const DOT_COLOR: Record<MonitorStatus, string> = {
  up: "bg-tertiary",
  down: "bg-primary",
  pending: "bg-muted-foreground/40",
};

/** `important` and `page` for the history table live in the URL. */
function useHistoryParams() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const important = searchParams.get("important") === "true";
  const page = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);

  const update = useCallback(
    (next: { important?: boolean; page?: number }) => {
      const params = new URLSearchParams(searchParams.toString());
      if (next.important !== undefined) {
        if (next.important) params.set("important", "true");
        else params.delete("important");
        params.delete("page");
      }
      if (next.page !== undefined) {
        if (next.page <= 1) params.delete("page");
        else params.set("page", String(next.page));
      }
      const query = params.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams],
  );

  return { important, page, update };
}

function Fact({ label, children, title }: { label: string; children: ReactNode; title?: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 truncate text-sm" title={title}>
        {children}
      </dd>
    </div>
  );
}

function dateOrDash(iso: string | null | undefined): string {
  return iso ? formatDateTime(iso) : "—";
}

function notificationSummary(monitor: Monitor): string {
  if (monitor.notifications.length === 0) return "None";
  const counts = new Map<string, number>();
  for (const n of monitor.notifications) {
    const label = notificationTypeLabel[n.type];
    counts.set(label, (counts.get(label) ?? 0) + 1);
  }
  const parts = [...counts.entries()].map(([label, count]) => (count > 1 ? `${label} ×${count}` : label));
  return `${monitor.notifications.length} · ${parts.join(", ")}`;
}

function Facts({ monitor }: { monitor: Monitor }) {
  return (
    <dl className="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3 lg:grid-cols-4">
      <Fact label="Interval">{formatInterval(monitor.interval_seconds)}</Fact>
      <Fact label="Timeout">{formatInterval(monitor.timeout_seconds)}</Fact>
      <Fact label="Retries">{monitor.retries}</Fact>
      <Fact label="Failures in a row">{monitor.failure_count}</Fact>
      <Fact label="Next check" title={dateOrDash(monitor.next_check_at)}>
        {monitor.active ? formatRelative(monitor.next_check_at) : "Paused"}
      </Fact>
      <Fact label="Last checked" title={dateOrDash(monitor.last_checked_at)}>
        {formatRelative(monitor.last_checked_at)}
      </Fact>
      <Fact label="Certificate expires">{dateOrDash(monitor.cert_expires_at)}</Fact>
      <Fact label="Domain expires">{dateOrDash(monitor.domain_expires_at)}</Fact>
      <Fact label="Method">{monitor.method}</Fact>
      <Fact label="Auth">{authMethodLabel[monitor.auth_method]}</Fact>
      <Fact label="Notifications">{notificationSummary(monitor)}</Fact>
    </dl>
  );
}

function StatusCell({ status }: { status: MonitorStatus }) {
  return (
    <span className="inline-flex items-center gap-2">
      <span aria-hidden="true" className={`size-2 rounded-full ${DOT_COLOR[status]}`} />
      {monitorStatusLabel[status]}
    </span>
  );
}

const history = createListColumnHelper<Heartbeat>();

const historyColumns = history.columns([
  history.accessor("checked_at", {
    header: "Checked at",
    cell: ({ getValue }) => formatDateTime(getValue()),
    meta: { className: "whitespace-nowrap", skeleton: <Skeleton className="h-4 w-32" /> },
  }),
  history.accessor("status", {
    header: "Status",
    cell: ({ getValue }) => <StatusCell status={getValue()} />,
    meta: { skeleton: <Skeleton className="h-4 w-12" /> },
  }),
  history.accessor("http_status", {
    header: "HTTP",
    cell: ({ getValue }) => getValue() || "—",
    meta: { className: "tabular-nums", skeleton: <Skeleton className="h-4 w-8" /> },
  }),
  history.accessor("latency_ms", {
    header: "Latency",
    cell: ({ getValue }) => formatLatency(getValue()),
    meta: { className: "tabular-nums", skeleton: <Skeleton className="h-4 w-12" /> },
  }),
  history.display({
    id: "message",
    header: "Message",
    cell: ({ row }) => {
      const heartbeat = row.original;
      return (
        <>
          <span className="block truncate" title={heartbeat.message}>
            {heartbeat.message || "—"}
          </span>
          {heartbeat.error_body ? (
            <details className="mt-1">
              <summary className="cursor-pointer text-xs text-muted-foreground underline-offset-4 hover:underline">
                Response
              </summary>
              <pre className="mt-2 max-h-64 overflow-auto rounded-md bg-muted p-3 font-mono text-xs whitespace-pre-wrap">
                {heartbeat.error_body}
              </pre>
            </details>
          ) : null}
        </>
      );
    },
    meta: { className: "max-w-md", skeleton: <Skeleton className="h-4 w-48" /> },
  }),
]);

function History({ id }: { id: string }) {
  const params = useHistoryParams();
  const offset = (params.page - 1) * MONITOR_PAGE_SIZE;
  const query = useQuery(
    heartbeatsQueryOptions(id, {
      important: params.important ? true : undefined,
      limit: MONITOR_PAGE_SIZE,
      offset,
    }),
  );
  const items = query.data?.items ?? NO_ROWS;
  const total = query.data?.meta.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / MONITOR_PAGE_SIZE));

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-base font-semibold">Check history</h2>
        <div className="flex items-center gap-2">
          <Switch
            id="history-important"
            checked={params.important}
            onCheckedChange={(checked) => params.update({ important: checked })}
          />
          <Label htmlFor="history-important" className="text-sm">
            Only status changes
          </Label>
        </div>
      </div>

      {query.isError ? (
        <ErrorPanel error={query.error} onRetry={() => query.refetch()} title="Could not load the check history" />
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border">
          <DataTable
            columns={historyColumns}
            data={items}
            isPending={query.isPending}
            emptyMessage={params.important ? "No status changes yet." : "No checks yet."}
          />
        </div>
      )}

      {query.isSuccess && pageCount > 1 ? (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-muted-foreground">
            Showing {offset + 1}–{Math.min(offset + items.length, total)} of {total}
          </p>
          <Pagination className="mx-0 w-auto">
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  href="#"
                  aria-disabled={params.page <= 1}
                  className={params.page <= 1 ? "pointer-events-none opacity-50" : undefined}
                  onClick={(event) => {
                    event.preventDefault();
                    params.update({ page: params.page - 1 });
                  }}
                />
              </PaginationItem>
              <PaginationItem>
                <span className="px-2 text-sm text-muted-foreground tabular-nums">
                  {params.page} / {pageCount}
                </span>
              </PaginationItem>
              <PaginationItem>
                <PaginationNext
                  href="#"
                  aria-disabled={params.page >= pageCount}
                  className={params.page >= pageCount ? "pointer-events-none opacity-50" : undefined}
                  onClick={(event) => {
                    event.preventDefault();
                    params.update({ page: params.page + 1 });
                  }}
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      ) : null}
    </section>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-8" aria-busy="true">
      <Skeleton className="h-4 w-20" />
      <div className="space-y-2">
        <Skeleton className="h-7 w-64" />
        <Skeleton className="h-4 w-80" />
      </div>
      <div className="grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-4">
        {Array.from({ length: 8 }, (_, i) => (
          <div key={i} className="space-y-1.5">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-4 w-24" />
          </div>
        ))}
      </div>
      <Skeleton className="h-12 w-full max-w-2xl" />
    </div>
  );
}

export function MonitorDetail({ id }: { id: string }) {
  const roles = useRoles();
  const writable = canWrite(roles);
  const deletable = canDelete(roles);
  const [confirming, setConfirming] = useState(false);

  const query = useQuery(monitorQueryOptions(id));
  const strip = useQuery({
    ...heartbeatsQueryOptions(id, { important: false, limit: STRIP_COUNT }),
    staleTime: 30_000,
    enabled: query.isSuccess,
  });
  const pauseResume = usePauseResume();
  const remove = useDeleteMonitor({ redirectTo: "/monitors" });

  const backLink = (
    <Link
      href="/monitors"
      className="inline-flex items-center gap-1.5 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
    >
      <ArrowLeftIcon className="size-4" aria-hidden="true" />
      Monitors
    </Link>
  );

  if (query.isPending) return <DetailSkeleton />;

  if (query.isError) {
    if (query.error instanceof ApiError && query.error.status === 404) {
      return (
        <div className="space-y-6">
          {backLink}
          <NotFoundPanel />
        </div>
      );
    }
    return (
      <div className="space-y-6">
        {backLink}
        <ErrorPanel error={query.error} onRetry={() => query.refetch()} title="Could not load this monitor" />
      </div>
    );
  }

  const monitor = query.data;

  return (
    <div className="space-y-8">
      <div className="space-y-4">
        {backLink}
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 space-y-2">
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="text-2xl font-semibold tracking-tight">{monitor.name}</h1>
              <MonitorStatusBadge monitor={monitor} />
            </div>
            <a
              href={monitor.url}
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex max-w-full items-center gap-1.5 font-mono text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
            >
              <span className="truncate">{monitor.url}</span>
              <ExternalLinkIcon className="size-3.5 shrink-0" aria-hidden="true" />
              <span className="sr-only">(opens in a new tab)</span>
            </a>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {writable ? (
              <Button
                disabled={pauseResume.isPending}
                onClick={() => pauseResume.mutate({ id: monitor.id, active: monitor.active })}
              >
                {monitor.active ? "Pause checks" : "Resume checks"}
              </Button>
            ) : null}
            <Button variant="outline" asChild>
              <Link href={`/monitors/${monitor.id}/edit`}>Edit</Link>
            </Button>
            {deletable ? (
              <Button variant="destructive" onClick={() => setConfirming(true)}>
                Delete
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      <Facts monitor={monitor} />

      <section className="space-y-3">
        <h2 className="text-base font-semibold">Last {STRIP_COUNT} checks</h2>
        {strip.isPending ? (
          <Skeleton className="h-12 w-full max-w-2xl" />
        ) : (
          <HeartbeatStrip
            heartbeats={strip.data?.items ?? []}
            count={STRIP_COUNT}
            size="lg"
            className="max-w-full overflow-x-auto"
          />
        )}
      </section>

      <History id={id} />

      <DeleteMonitorDialog
        open={confirming}
        onOpenChange={setConfirming}
        pending={remove.isPending}
        onConfirm={() => remove.mutate(monitor.id)}
      />
    </div>
  );
}
