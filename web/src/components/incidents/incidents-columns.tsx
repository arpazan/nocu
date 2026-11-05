"use client";

import Link from "next/link";

import { SeverityMark } from "@/components/incidents/severity-badge";
import { IncidentStatusBadge } from "@/components/incidents/status-badge";
import { createListColumnHelper } from "@/components/list/data-table";
import { Skeleton } from "@/components/ui/skeleton";
import type { Incident } from "@/lib/api/types";
import { formatDateTime, formatRelative } from "@/lib/format";

/** Source and opening time leave the table below the wide layout and sit under the title instead. */
const LG_UP = "hidden lg:table-cell";

/** "1 min ago · snmp" for narrow screens, where those columns are hidden. */
function RowMeta({ incident }: { incident: Incident }) {
  return (
    <p className="mt-0.5 text-xs text-muted-foreground lg:hidden">
      <time dateTime={incident.created_at} title={formatDateTime(incident.created_at)}>
        {formatRelative(incident.created_at)}
      </time>
      {incident.source ? <span className="font-mono"> · {incident.source}</span> : null}
    </p>
  );
}

const helper = createListColumnHelper<Incident>();

export const incidentColumns = helper.columns([
  helper.accessor("severity", {
    header: "Severity",
    cell: ({ getValue }) => <SeverityMark severity={getValue()} />,
    meta: { className: "md:w-28", skeleton: <Skeleton className="h-4 w-16" /> },
  }),
  helper.accessor("title", {
    header: "Title",
    cell: ({ row }) => {
      const incident = row.original;
      return (
        <>
          <Link
            href={`/incidents/${incident.id}`}
            className="font-medium underline-offset-4 hover:underline focus-visible:underline"
          >
            {incident.title}
          </Link>
          {incident.description && (
            <p className="mt-0.5 truncate text-xs text-muted-foreground">{incident.description}</p>
          )}
          <RowMeta incident={incident} />
        </>
      );
    },
    meta: { className: "w-full max-w-0 whitespace-normal", skeleton: <Skeleton className="h-4 w-64" /> },
  }),
  helper.accessor("status", {
    header: "Status",
    cell: ({ getValue }) => <IncidentStatusBadge status={getValue()} />,
    meta: { skeleton: <Skeleton className="h-5 w-20 rounded-4xl" /> },
  }),
  helper.accessor("source", {
    header: "Source",
    cell: ({ getValue }) => getValue() || "—",
    meta: { className: `${LG_UP} font-mono text-xs text-muted-foreground`, skeleton: <Skeleton className="h-4 w-12" /> },
  }),
  helper.accessor("created_at", {
    header: "Opened",
    cell: ({ getValue }) => (
      <time dateTime={getValue()} title={formatDateTime(getValue())}>
        {formatRelative(getValue())}
      </time>
    ),
    meta: { className: `${LG_UP} text-muted-foreground`, skeleton: <Skeleton className="h-4 w-16" /> },
  }),
]);
