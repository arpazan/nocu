# Nocu web app

Next.js (App Router) UI for Nocu. Tailwind v4 and shadcn/ui on Radix. Member of the pnpm workspace
defined in the root `pnpm-workspace.yaml`.

[← Back to README](../README.md)

## Requirements

- Node 24 (see [`.nvmrc`](.nvmrc))
- pnpm (`corepack enable && corepack prepare pnpm@latest --activate`)
- identity on `:8081` and core-api on `:8080`, e.g. `docker compose up -d postgres nats` plus `make dev`
  from the repo root (see the [development guide](../docs/development.md))

## Commands

Run from the repository root:

```bash
make web-install                     # pnpm install --frozen-lockfile
make dev-web                         # next dev on http://localhost:3000
make web-lint && make web-test       # eslint + tsc, vitest
make web-build                       # next build (standalone output)
```

Or from `web/` directly with `pnpm dev`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build`.

## Configuration

With no `.env.local` the app targets `localhost:8081` (identity) and `localhost:8080` (core-api).
Copy [`.env.example`](.env.example) to `.env.local` to point elsewhere.

`NEXT_PUBLIC_*` values are baked into the browser bundle at build time, so `build/Dockerfile.web`
takes them as build args and the compose file passes them through from the root `.env`.

## Structure

```
src/app/
  (app)/            authenticated pages: overview, incidents, monitors
  login/, register/ public pages
  api/auth/*        route handlers that proxy identity and keep the refresh token in an httpOnly cookie
src/components/     app shell, auth forms, feature components (incidents/, monitors/, overview/), shadcn under ui/
src/lib/api/        typed fetch client for the {data, meta} / {error} envelopes and TanStack Query options
src/lib/auth/       zod schemas, in-memory access-token store, session bootstrap and role helpers
src/lib/realtime/   WebSocket event stream hook and store
src/lib/query-client.ts  QueryClient defaults
```

## Conventions

**UI components.** Add shadcn components with `pnpm dlx shadcn@latest add <name>`; they land in
`src/components/ui/`.

**Data fetching.** [TanStack Query](https://tanstack.com/query). `QueryProvider` in the root layout
owns one `QueryClient` per browser tab (defaults in `src/lib/query-client.ts`, which never retries 4xx
responses), and devtools are available in development. Define queries with `queryOptions` next to the
API code, as `src/lib/api/health.ts` does.

**Forms.** [react-hook-form](https://react-hook-form.com) with [zod](https://zod.dev) schemas via
`zodResolver`, rendered with shadcn's `Field` components (shadcn v4 replaced the old `Form` wrapper
with `Field` plus `Controller`). `src/components/login-form.tsx` is the reference, including mapping
backend `ApiError`s onto form errors.

**Auth.** The access token returned by `/api/auth/login` is kept in memory in
`src/lib/auth/session-store.ts`; a page reload gets a new one from `POST /api/auth/refresh`, which
reads the httpOnly refresh cookie.

**Realtime.** `src/lib/realtime/use-event-stream.ts` connects to core-api's `GET /api/v1/ws` with the
access token and feeds events into the query cache.

## Screenshots

See [docs/screenshots.md](../docs/screenshots.md).
