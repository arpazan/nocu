"use client";

import { useQuery } from "@tanstack/react-query";
import { MoreHorizontalIcon } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { createListColumnHelper } from "@/components/list/data-table";
import { HeartbeatStrip } from "@/components/monitors/heartbeat-strip";
import { DeleteMonitorDialog, useDeleteMonitor, usePauseResume } from "@/components/monitors/monitor-mutations";
import { MonitorStatusBadge } from "@/components/monitors/monitor-status-badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { heartbeatsQueryOptions } from "@/lib/api/monitors";
import type { Monitor } from "@/lib/api/types";
import { formatDateTime, formatInterval, formatRelative } from "@/lib/format";

const STRIP_COUNT = 30;

/*
 * Column visibility by viewport. Narrow screens keep name, status and actions;
 * the interval and last check move into a line under the URL. The heartbeat
 * strip needs 180px of its own, so it waits for the wide layout.
 */
const MD_UP = "hidden md:table-cell";
const LG_UP = "hidden lg:table-cell";

/** The last 30 probes for one row; refreshed at most every 30 seconds. */
function RowHeartbeats({ id }: { id: string }) {
  const query = useQuery({
    ...heartbeatsQueryOptions(id, { important: false, limit: STRIP_COUNT }),
    staleTime: 30_000,
  });
  if (query.isPending) return <Skeleton className="h-6 w-[11.25rem]" />;
  return <HeartbeatStrip heartbeats={query.data?.items ?? []} count={STRIP_COUNT} />;
}

function RowActions({ monitor, writable, deletable }: { monitor: Monitor; writable: boolean; deletable: boolean }) {
  const [confirming, setConfirming] = useState(false);
  const pauseResume = usePauseResume();
  const remove = useDeleteMonitor();

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon-sm" aria-label={`Actions for ${monitor.name}`}>
            <MoreHorizontalIcon />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {writable ? (
            <DropdownMenuItem
              disabled={pauseResume.isPending}
              onSelect={() => pauseResume.mutate({ id: monitor.id, active: monitor.active })}
            >
              {monitor.active ? "Pause checks" : "Resume checks"}
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem asChild>
            <Link href={`/monitors/${monitor.id}/edit`}>Edit</Link>
          </DropdownMenuItem>
          {deletable ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onSelect={() => setConfirming(true)}>
                Delete
              </DropdownMenuItem>
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
      <DeleteMonitorDialog
        open={confirming}
        onOpenChange={setConfirming}
        pending={remove.isPending}
        onConfirm={() => remove.mutate(monitor.id, { onSuccess: () => setConfirming(false) })}
      />
    </>
  );
}

/** "Checked 2 min ago · every 60 s" for narrow screens, where those columns are hidden. */
function RowMeta({ monitor }: { monitor: Monitor }) {
  const checked = monitor.last_checked_at ? `Checked ${formatRelative(monitor.last_checked_at)}` : "Not checked yet";
  return (
    <p className="mt-0.5 text-xs text-muted-foreground md:hidden">
      {checked} · every {formatInterval(monitor.interval_seconds)}
    </p>
  );
}

const helper = createListColumnHelper<Monitor>();

/**
 * The actions column depends on the viewer's roles, so the column set is
 * built per table rather than shared; memoise the result on those flags.
 */
export function monitorColumns({ writable, deletable }: { writable: boolean; deletable: boolean }) {
  return helper.columns([
    helper.accessor("name", {
      header: "Name",
      cell: ({ row }) => {
        const monitor = row.original;
        return (
          <>
            <Link
              href={`/monitors/${monitor.id}`}
              className="block truncate font-medium underline-offset-4 hover:underline focus-visible:underline"
              title={monitor.name}
            >
              {monitor.name}
            </Link>
            <p className="truncate font-mono text-xs text-muted-foreground" title={monitor.url}>
              {monitor.url}
            </p>
            <RowMeta monitor={monitor} />
          </>
        );
      },
      meta: {
        className: "w-full max-w-0",
        skeleton: (
          <>
            <Skeleton className="h-4 w-40" />
            <Skeleton className="mt-1.5 h-3 w-56" />
          </>
        ),
      },
    }),
    helper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => <MonitorStatusBadge monitor={row.original} />,
      meta: { skeleton: <Skeleton className="h-5 w-12 rounded-4xl" /> },
    }),
    helper.display({
      id: "heartbeats",
      header: "Recent checks",
      cell: ({ row }) => <RowHeartbeats id={row.original.id} />,
      meta: { className: LG_UP, skeleton: <Skeleton className="h-6 w-[11.25rem]" /> },
    }),
    helper.accessor("interval_seconds", {
      header: "Interval",
      cell: ({ getValue }) => formatInterval(getValue()),
      meta: { className: `${MD_UP} tabular-nums`, skeleton: <Skeleton className="h-4 w-10" /> },
    }),
    helper.accessor("last_checked_at", {
      header: "Last checked",
      cell: ({ getValue }) => <span title={formatDateTime(getValue())}>{formatRelative(getValue())}</span>,
      meta: { className: `${MD_UP} text-muted-foreground`, skeleton: <Skeleton className="h-4 w-16" /> },
    }),
    helper.display({
      id: "actions",
      header: () => <span className="sr-only">Actions</span>,
      cell: ({ row }) => <RowActions monitor={row.original} writable={writable} deletable={deletable} />,
      meta: { className: "w-12 text-right", skeleton: null },
    }),
  ]);
}
