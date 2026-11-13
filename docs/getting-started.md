# Getting started

Bring the whole stack up with Docker Compose and walk through the API end to end.

[← Back to README](../README.md)

## 1. Start the stack

```bash
cp .env.example .env
docker compose up --build -d        # postgres, nats, migrate, the four Go services and web on :3000
```

The web app is now at <http://localhost:3000>, identity at `:8081` and core-api at `:8080`.
Every setting lives in `.env`; see [`.env.example`](../.env.example) for the full list with comments.

## 2. Register and log in

```bash
curl -s localhost:8081/api/v1/auth/register -H 'content-type: application/json' \
  -d '{"email":"ops@example.com","password":"correct horse","name":"Ops"}'

TOKEN=$(curl -s localhost:8081/api/v1/auth/login -H 'content-type: application/json' \
  -d '{"email":"ops@example.com","password":"correct horse"}' | jq -r .data.access_token)
```

New users get the `viewer` role, which can read incidents and monitors and connect to the WebSocket.
Creating or updating incidents and monitors requires `operator` or `admin`; grant it directly in the
`user_roles` table for now, then log in again.

## 3. Watch the event stream

In another terminal (needs [websocat](https://github.com/vi/websocat)):

```bash
websocat "ws://localhost:8080/api/v1/ws?token=$TOKEN"
```

## 4. Create an incident

```bash
curl -s localhost:8080/api/v1/incidents -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{"title":"Core router down","severity":"critical","source":"snmp"}'
```

The WebSocket client receives an `incident.created` frame, and the consumer stores the same message as
an `events` row.

## Next steps

- [Uptime monitors](monitors.md) — probe URLs on an interval and get notified on status changes.
- [API reference (Bruno)](../bruno/README.md) — every endpoint with examples.
- [Development guide](development.md) — run services with hot reload instead of Compose.
