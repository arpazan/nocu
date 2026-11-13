# Uptime monitors

A monitor probes one `http://` or `https://` URL on a fixed interval. The scheduler picks up monitors
that are due, runs the request with the configured method, body, headers, authentication, redirect
limit and timeout, stores a heartbeat per probe, and publishes `monitor.up` / `monitor.down` when the
status changes. Optional daily checks publish `monitor.cert_expiring` and `monitor.domain_expiring`.

[← Back to README](../README.md)

## Create a monitor

Requires an access token with the `operator` or `admin` role (see [Getting started](getting-started.md)).

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

Within the first interval a WebSocket client receives a `monitor.up` (or `monitor.down`) frame; later
frames arrive only when the status flips.

## Endpoints

| Method and path | Purpose |
|-----------------|---------|
| `GET /api/v1/monitors` | list monitors |
| `POST /api/v1/monitors` | create |
| `GET /api/v1/monitors/{id}` | get one |
| `PATCH /api/v1/monitors/{id}` | update |
| `DELETE /api/v1/monitors/{id}` | delete |
| `GET /api/v1/monitors/{id}/heartbeats` | probe history |
| `POST /api/v1/monitors/{id}/pause` | stop probing |
| `POST /api/v1/monitors/{id}/resume` | restart probing |

See [`bruno/core-api/monitors`](../bruno/core-api/monitors) for the full field list.

## Auto-created incidents

With `"auto_create_incident": true` the scheduler opens a `critical` incident (source `monitor`,
linked by `monitor_id`) the moment the monitor flips to `down`, and publishes `incident.created`.
A monitor never has more than one open or acknowledged incident at a time; recovery does not close it,
an operator does. `GET /api/v1/incidents?monitor_id={id}` lists a monitor's incidents.

## Notification channels

Channels are attached per monitor in the `notifications` array.

| Type      | Delivered | Notes |
|-----------|-----------|-------|
| `discord` | yes       | webhook URL, bot display name, prefix message, `text` or `embed` format, optional URL hiding |
| `webhook` | yes       | post URL, method, `json` / `form` / `custom` body with placeholders, extra headers |
| `webpush` | not yet   | the browser `PushSubscription` is stored and validated; delivery needs VAPID keys and is a follow-up |

## Security notes

- Secrets (`basic_password`, `bearer_token`) are write-only and never returned by the API.
- Probes refuse loopback and private IP targets unless `SCHEDULER_MONITOR_ALLOW_PRIVATE=true`.

## Scheduler settings

Tuning knobs from [`.env.example`](../.env.example):

| Variable | Default | Meaning |
|----------|---------|---------|
| `SCHEDULER_MONITOR_TICK` | `10s` | how often the scheduler looks for monitors due for a probe |
| `SCHEDULER_MONITOR_CONCURRENCY` | `10` | max probes run in parallel per tick |
| `SCHEDULER_MONITOR_ALLOW_PRIVATE` | `false` | allow loopback/private IP targets (SSRF guard) |
| `SCHEDULER_HEARTBEAT_RETENTION` | `720h` | how long heartbeats are kept before cleanup |
| `SCHEDULER_EXPIRY_WARN_DAYS` | `7,14,30` | days before certificate/domain expiry at which a warning is published |
