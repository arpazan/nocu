import { queryOptions } from "@tanstack/react-query";

import { ApiClient } from "@/lib/api/client";
import type { Health } from "@/lib/api/types";
import { env } from "@/lib/env";

export const healthTargets = [
  { name: "identity", url: env.identityUrl },
  { name: "core-api", url: env.coreApiUrl },
] as const;

export type HealthTarget = (typeof healthTargets)[number];

/** Query options for one service's /healthz. Throws ApiError on non-2xx. */
export function healthQueryOptions(target: HealthTarget) {
  return queryOptions({
    queryKey: ["health", target.name] as const,
    queryFn: async () => {
      const client = new ApiClient(target.url);
      const health = await client.raw<Health>("/healthz", { cache: "no-store" });
      if (health.status !== "ok") throw new Error(`status=${health.status}`);
      return health;
    },
    staleTime: 10 * 1000,
    refetchInterval: 30 * 1000,
    retry: false,
  });
}
