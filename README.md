# Nocu

A NOC platform in one repo: an incident and uptime-monitor API with realtime WebSocket push, an
identity service, an event consumer and a cron scheduler that probes monitors, plus a Next.js web
app. Go services use PostgreSQL via GORM and NATS JetStream for messaging.

![Nocu web app](docs/screenshots/Artboard.png)

## Services

| Service   | Port | Role |
|-----------|------|------|
| identity  | 8081 | auth, users, JWTs |
| core-api  | 8080 | incidents, monitors, WebSocket event stream |
| consumer  |      | persists every JetStream message as an event |
| scheduler |      | cron jobs, uptime probes, expiry checks, notifications |
| web       | 3000 | Next.js UI |

Details in [docs/architecture.md](docs/architecture.md).

## Quick start

```bash
cp .env.example .env
docker compose up --build -d        # postgres, nats, migrate, the four Go services and web on :3000
```

Open <http://localhost:3000>, register, and log in. For the API flow with `curl` and a WebSocket
client, follow [docs/getting-started.md](docs/getting-started.md).

For local development with hot reload:

```bash
docker compose up -d postgres nats
make migrate
make dev            # Go services (Air)
make dev-web        # Next.js on :3000
```

## Documentation

| Topic | Where |
|-------|-------|
| Architecture, services, repository layout, roles | [docs/architecture.md](docs/architecture.md) |
| Getting started: compose, register, incidents, WebSocket | [docs/getting-started.md](docs/getting-started.md) |
| Uptime monitors: probes, notifications, auto-incidents | [docs/monitors.md](docs/monitors.md) |
| Development: hot reload, migrations, tests, Docker images | [docs/development.md](docs/development.md) |
| Web app: setup, structure, conventions | [web/README.md](web/README.md) |
| API reference (Bruno collection) | [bruno/README.md](bruno/README.md) |
| Screenshots | [docs/screenshots.md](docs/screenshots.md) |
| Configuration reference | [.env.example](.env.example) |
