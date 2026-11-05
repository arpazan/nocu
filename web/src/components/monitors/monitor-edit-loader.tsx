"use client";

import { useQuery } from "@tanstack/react-query";

import { MonitorForm } from "@/components/monitors/monitor-form";
import { ErrorPanel, NotFoundPanel } from "@/components/monitors/monitor-panels";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError } from "@/lib/api/client";
import { monitorQueryOptions } from "@/lib/api/monitors";

/** Loads one monitor and hands it to the form; the form keeps its own state once mounted. */
export function MonitorEditLoader({ id }: { id: string }) {
  const query = useQuery(monitorQueryOptions(id));

  if (query.isPending) {
    return (
      <div className="max-w-3xl space-y-6" aria-busy="true">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-8 w-full" />
          </div>
        ))}
      </div>
    );
  }

  if (query.isError) {
    if (query.error instanceof ApiError && query.error.status === 404) return <NotFoundPanel />;
    return <ErrorPanel error={query.error} onRetry={() => query.refetch()} title="Could not load this monitor" />;
  }

  return <MonitorForm monitor={query.data} />;
}
