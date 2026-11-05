/**
 * Browser-visible configuration. Only NEXT_PUBLIC_* variables are inlined into
 * the client bundle, and each must be referenced by its literal name for the
 * bundler to pick it up. Every value falls back to the docker-compose host
 * ports so `pnpm dev` and `pnpm build` work with no .env file; deployments set
 * them explicitly (build/Dockerfile.web takes them as build args).
 */

function read(value: string | undefined, fallback: string): string {
  const v = value && value.length > 0 ? value : fallback;
  return v.replace(/\/+$/, "");
}

export const env = {
  /** identity service base URL (register, login, refresh, /me). */
  identityUrl: read(process.env.NEXT_PUBLIC_IDENTITY_URL, "http://localhost:8081"),
  /** core-api base URL (incidents, monitors). */
  coreApiUrl: read(process.env.NEXT_PUBLIC_CORE_API_URL, "http://localhost:8080"),
  /** WebSocket endpoint for the realtime event feed. */
  wsUrl: read(process.env.NEXT_PUBLIC_WS_URL, "ws://localhost:8080/api/v1/ws"),
} as const;
