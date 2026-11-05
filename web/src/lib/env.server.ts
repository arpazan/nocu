import "server-only";

import { env } from "@/lib/env";

/**
 * Server-side configuration. Route handlers talk to the Go services over the
 * Docker network (http://identity:8081) while the browser uses the host ports,
 * so the server-to-server URLs are separate, non-public variables that default
 * to the public ones for local development.
 */
export const serverEnv = {
  identityUrl: (process.env.IDENTITY_URL ?? env.identityUrl).replace(/\/+$/, ""),
  coreApiUrl: (process.env.CORE_API_URL ?? env.coreApiUrl).replace(/\/+$/, ""),
} as const;
