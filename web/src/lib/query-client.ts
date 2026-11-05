import { QueryClient, isServer } from "@tanstack/react-query";

import { ApiError } from "@/lib/api/client";

/** Retrying a 4xx never helps; retry only network failures and 5xx, up to twice. */
function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status < 500) return false;
  return failureCount < 2;
}

export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Above 0 so data fetched during SSR is not immediately refetched on the client.
        staleTime: 30 * 1000,
        retry: shouldRetry,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

/**
 * One client per browser tab, a fresh one per server request. Reusing the
 * browser client across re-renders keeps the cache alive when React suspends
 * during the initial render.
 */
export function getQueryClient(): QueryClient {
  if (isServer) return makeQueryClient();
  browserQueryClient ??= makeQueryClient();
  return browserQueryClient;
}
