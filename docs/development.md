# Development guide

Running the Go services locally with hot reload, migrations and Docker images.
For the Next.js app see the [web app README](../web/README.md).

[← Back to README](../README.md)

## Prerequisites

- Go 1.26, Docker, `make`
- [Air](https://github.com/air-verse/air) for hot reload: `go install github.com/air-verse/air@latest`
- Node 24 and pnpm for the web app (see [web/README.md](../web/README.md))

## Daily loop

```bash
docker compose up -d postgres nats   # just the dependencies
make migrate                         # apply migrations
make dev                             # all four services with hot reload (Air)
make test                            # unit tests (Go + web)
make lint                            # gofmt + go vet + web lint/typecheck
```

`make help` lists every target.

## Hot reload

`make dev-<service>` runs one service under Air and rebuilds it whenever a `.go` file under
`services/<service>/` or `shared/` changes. `make dev` runs all four in one terminal; Ctrl+C stops
everything.

Configs live in `.air/<service>.toml`. Identity is pinned to port 8081 there so it never collides
with core-api. Build output goes to `tmp/`, which is gitignored.

Without Air, `HTTP_PORT=8081 make run-identity` and `make run-<service>` still work but need a
manual restart after each change.

## Configuration

All services read their settings from the environment. `make` loads `.env` automatically when present.
[`.env.example`](../.env.example) documents every variable; copy it to `.env` to start.

## Migrations

Migrations live in [`shared/migration`](../shared/migration) and run with gormigrate inside a transaction.

```bash
make migrate          # apply pending migrations
make migrate-down     # roll back the last migration
make migrate-status   # show migration state
```

- Add a new file `shared/migration/<timestamp>_<name>.go` and append it to `list` in `migration.go`.
- Only the init migration references `shared/model` directly. Later migrations declare local struct
  copies so editing a model never rewrites history.

## Testing

```bash
make test        # go test ./... + vitest
make test-race   # with the race detector
make cover       # coverage summary
```

## Docker images

```bash
make docker-build                 # all six images
make docker-build-core-api        # one image
make up / make down / make logs   # compose shortcuts
```

Each Go Dockerfile in [`build/`](../build) is a two-stage build: `golang:1.26-alpine` compiles a static
binary, and `gcr.io/distroless/static-debian12:nonroot` runs it. `Dockerfile.web` installs the pnpm
workspace, runs `next build` with `output: "standalone"`, and serves it from `node:24-alpine` as the
`node` user. The build context must be the repository root.

Images are tagged `$(REGISTRY)/<name>:$(TAG)`; override with `make docker-build REGISTRY=ghcr.io/acme TAG=v1`.
