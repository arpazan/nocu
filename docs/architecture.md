# Architecture

Nocu is a monorepo of four Go services, a migration CLI and a Next.js web app. Go services share
PostgreSQL (via GORM) for state and NATS JetStream for messaging.

[← Back to README](../README.md)

## Services

| Service   | Port | Role |
|-----------|------|------|
| identity  | 8081 | register, login, refresh, `/me`; issues JWTs |
| core-api  | 8080 | incident CRUD, monitor CRUD and heartbeat history; `GET /api/v1/ws` pushes every `incident.*`, `job.*` and `monitor.*` event to clients |
| consumer  |      | durable JetStream consumer; persists every message as an `events` row |
| scheduler |      | cron jobs (heartbeat, SLA check, event cleanup, heartbeat cleanup), uptime probes and certificate/domain expiry checks, with Postgres advisory locks |
| web       | 3000 | Next.js UI; talks to identity and core-api from the browser, so their `CORS_ORIGINS` must include the web origin outside development |

## Event flow

1. core-api (or the scheduler) publishes `incident.*`, `job.*` and `monitor.*` subjects to JetStream.
2. consumer durably subscribes and writes every message to the `events` table.
3. core-api fans the same subjects out to WebSocket clients on `GET /api/v1/ws`.

## Repository layout

```
build/              one Dockerfile per image (Dockerfile.core-api, .identity, .consumer, .scheduler, .migrate, .web)
cmd/migrate/        migration CLI (up | down | status)
services/<name>/
  cmd/main.go       service entrypoint
  internal/         service-private packages
                    core-api/internal/monitor: monitor CRUD and heartbeat history
                    scheduler/internal/uptime: HTTP probes, expiry checks, notification delivery
shared/
  auth/             JWT issue and verify
  config/           env-based config blocks (Base, HTTP, JWT)
  database/         GORM + PostgreSQL connection, slog bridge
  httpx/            Echo server factory, middleware, error envelope
  logger/           slog setup
  messaging/        Publisher/Subscriber contract, NATS JetStream and in-memory impls
  migration/        versioned gormigrate migrations
  model/            GORM models (User, Role, Incident, Event, ScheduledJob, Monitor, MonitorNotification, MonitorHeartbeat)
web/                Next.js app (App Router, Tailwind v4, shadcn/ui on Radix); pnpm workspace member
  src/app/          routes: (app)/ overview, incidents, monitors; /login, /register;
                    /api/auth/* route handlers proxy identity and keep the refresh token in an httpOnly cookie
  src/components/   app shell, theme toggle, auth forms, feature components, shadcn components under ui/
  src/lib/api/      typed fetch client for the {data, meta} / {error} envelopes, TanStack Query options
  src/lib/auth/     zod schemas, in-memory access-token store, server-side session helpers
  src/lib/realtime/ WebSocket event stream hook and store
bruno/              Bruno collection documenting every endpoint
docs/               this documentation and screenshots
```

## Roles

New users get the `viewer` role, which can read incidents and monitors and connect to the WebSocket.
Creating or updating incidents and monitors requires `operator` or `admin`; grant it directly in the
`user_roles` table for now, then log in again.

## Related

- [Getting started](getting-started.md)
- [Uptime monitors](monitors.md)
- [Development guide](development.md)
- [Web app](../web/README.md)
