"use client";

import { useQueries } from "@tanstack/react-query";
import { RefreshCwIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError } from "@/lib/api/client";
import { healthQueryOptions, healthTargets } from "@/lib/api/health";

function describe(error: unknown): string {
  if (error instanceof ApiError) return `${error.status} ${error.message}`;
  return error instanceof Error ? error.message : "unreachable";
}

/**
 * Smoke check run from the browser: proves the NEXT_PUBLIC_* URLs, CORS on the
 * Go services, TanStack Query and the shadcn component set all work end to end.
 */
export function BackendHealth() {
  const results = useQueries({ queries: healthTargets.map((t) => healthQueryOptions(t)) });
  const refetching = results.some((r) => r.isFetching);

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="space-y-1">
          <CardTitle>Backend services</CardTitle>
          <CardDescription>Live /healthz checks from this browser, refreshed every 30s.</CardDescription>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={refetching}
          onClick={() => results.forEach((r) => void r.refetch())}
        >
          <RefreshCwIcon className={refetching ? "size-4 animate-spin" : "size-4"} />
          Refresh
        </Button>
      </CardHeader>
      <CardContent>
        <ul className="divide-y">
          {healthTargets.map((t, i) => {
            const r = results[i];
            return (
              <li key={t.name} className="flex items-center justify-between gap-4 py-3">
                <div className="min-w-0">
                  <p className="font-medium">{t.name}</p>
                  <p className="truncate font-mono text-xs text-muted-foreground">{t.url}</p>
                </div>
                {r.isPending && <Skeleton className="h-5 w-16" />}
                {r.isSuccess && <Badge>up</Badge>}
                {r.isError && (
                  <Badge variant="destructive" title={describe(r.error)}>
                    down
                  </Badge>
                )}
              </li>
            );
          })}
        </ul>
      </CardContent>
    </Card>
  );
}
