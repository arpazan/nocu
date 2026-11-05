# Nocu

Go monorepo for a NOC platform: an incident and uptime-monitor API with realtime WebSocket push, an identity service, an event consumer and a cron scheduler that probes monitors. PostgreSQL via GORM, NATS JetStream for messaging.

## Layout

```
build/              one Dockerfile per image (Dockerfile.core-api, .identity, .consumer, .scheduler, .migrate)
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
```

## Services

| Service   | Port | Role |
|-----------|------|------|
| identity  | 8081 | register, login, refresh, `/me`; issues JWTs |
| core-api  | 8080 | incident CRUD, monitor CRUD and heartbeat history; `GET /api/v1/ws` pushes every `incident.*`, `job.*` and `monitor.*` event to clients |
| consumer  |      | durable JetStream consumer; persists every message as an `events` row |
| scheduler |      | cron jobs (heartbeat, SLA check, event cleanup, heartbeat cleanup), uptime probes and certificate/domain expiry checks, with Postgres advisory locks |

## Quick start

```bash
cp .env.example .env
docker compose up --build -d        # postgres, nats, migrate, then all four services
```

Register, log in, create an incident, and watch it arrive over WebSocket:

```bash
curl -s localhost:8081/api/v1/auth/register -H 'content-type: application/json' \
  -d '{"email":"ops@example.com","password":"correct horse","name":"Ops"}'

TOKEN=$(curl -s localhost:8081/api/v1/auth/login -H 'content-type: application/json' \
  -d '{"email":"ops@example.com","password":"correct horse"}' | jq -r .data.access_token)

# in another terminal (needs websocat):
websocat "ws://localhost:8080/api/v1/ws?token=$TOKEN"

curl -s localhost:8080/api/v1/incidents -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{"title":"Core router down","severity":"critical","source":"snmp"}'
```

New users get the `viewer` role, which can read incidents and monitors and connect to the WebSocket. Creating or updating incidents and monitors requires `operator` or `admin`; grant it directly in the `user_roles` table for now, then log in again.

## Uptime monitors

A monitor probes one `http://` or `https://` URL on a fixed interval. The scheduler picks up monitors that are due, runs the request with the configured method, body, headers, authentication, redirect limit and timeout, stores a heartbeat per probe, and publishes `monitor.up` / `monitor.down` when the status changes. Optional daily checks publish `monitor.cert_expiring` and `monitor.domain_expiring`.

```bash
curl -s localhost:8080/api/v1/monitors -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{
    "name": "Public website",
    "url": "https://example.com/health",
    "interval_seconds": 60,
    "retries": 1,
    "timeout_seconds": 30,
    "cert_expiry_notification": true,
    "notifications": [
      { "type": "discord", "enabled": true,
        "config": { "webhook_url": "https://discord.com/api/webhooks/123/abc", "message_format": "embed" } }
    ]
  }'
```

Within the first interval the WebSocket client from the previous section receives a `monitor.up` (or `monitor.down`) frame; later frames arrive only when the status flips. `GET /api/v1/monitors/{id}/heartbeats` returns the probe history. `POST .../pause` and `.../resume` stop and restart probing.

Notification channels are attached per monitor:

| Type      | Delivered | Notes |
|-----------|-----------|-------|
| `discord` | yes       | webhook URL, bot display name, prefix message, `text` or `embed` format, optional URL hiding |
| `webhook` | yes       | post URL, method, `json` / `form` / `custom` body with placeholders, extra headers |
| `webpush` | not yet   | the browser `PushSubscription` is stored and validated; delivery needs VAPID keys and is a follow-up |

Secrets (`basic_password`, `bearer_token`) are write-only and never returned by the API. Probes refuse loopback and private IP targets unless `SCHEDULER_MONITOR_ALLOW_PRIVATE=true`. See `bruno/core-api/monitors` for the full field list.

## Local development

```bash
docker compose up -d postgres nats   # just the dependencies
make migrate                         # apply migrations
make dev                             # all four services with hot reload (Air)
make test                            # unit tests
make lint                            # gofmt + go vet
```

### Hot reload

`make dev-<service>` runs one service under [Air](https://github.com/air-verse/air) and rebuilds it whenever a `.go` file under `services/<service>/` or `shared/` changes. `make dev` runs all four in one terminal; Ctrl+C stops everything. Install Air once with:

```bash
go install github.com/air-verse/air@latest
```

Configs live in `.air/<service>.toml`. Identity is pinned to port 8081 there so it never collides with core-api. Build output goes to `tmp/`, which is gitignored.

Without Air, `HTTP_PORT=8081 make run-identity` and `make run-<service>` still work but need a manual restart after each change.

## Migrations

Migrations live in `shared/migration` and run with gormigrate inside a transaction.

- `make migrate` / `make migrate-down` / `make migrate-status`
- Add a new file `shared/migration/<timestamp>_<name>.go` and append it to `list` in `migration.go`.
- Only the init migration references `shared/model` directly. Later migrations declare local struct copies so editing a model never rewrites history.

## Docker images

```bash
make docker-build                 # all five images
make docker-build-core-api        # one image
```

Each Dockerfile is a two-stage build: `golang:1.26-alpine` compiles a static binary, and `gcr.io/distroless/static-debian12:nonroot` runs it. The build context must be the repository root.

## API documentation (Bruno)

The `bruno/` folder is a [Bruno](https://www.usebruno.com) collection that documents every endpoint. Open Bruno, choose **Open Collection**, pick `bruno/`, and select the `local` environment.

Run `identity / Register` then `identity / Login`; the login script stores the tokens in the environment and every other request inherits the bearer token. `core-api / incidents / Create` stores the new id in `incidentId` for the get, update, and delete requests. The `WebSocket (reference)` request documents the stream; connect with `websocat` or a browser client as described in its docs tab.
