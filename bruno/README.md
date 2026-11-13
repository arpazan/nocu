# API reference (Bruno)

This folder is a [Bruno](https://www.usebruno.com) collection that documents every endpoint of
identity and core-api, with request examples and docs tabs.

[← Back to README](../README.md)

## Open the collection

1. Open Bruno and choose **Open Collection**.
2. Pick this `bruno/` folder.
3. Select the `local` environment (identity on `:8081`, core-api on `:8080`).

## Walkthrough

1. Run `identity / Register`, then `identity / Login`. The login script stores the tokens in the
   environment, and every other request inherits the bearer token.
2. `core-api / incidents / Create` stores the new id in `incidentId` for the get, update, acknowledge,
   resolve, close and delete requests.
3. `core-api / monitors / Create` does the same for the monitor requests, including `heartbeats`,
   `pause` and `resume`.
4. The `WebSocket (reference)` request (`core-api / Ws`) documents the realtime stream; connect with
   `websocat` or a browser client as described in its docs tab.

## Folders

| Folder | Contents |
|--------|----------|
| `health/` | `healthz` and `readyz` for identity and core-api |
| `identity/` | register, login, refresh, me |
| `core-api/incidents/` | list, create, get, acknowledge, resolve, close, delete |
| `core-api/monitors/` | list, create, get, update, delete, heartbeats, pause, resume |
| `core-api/Ws.bru` | WebSocket reference |

## Related

- [Getting started](../docs/getting-started.md) — the same flow with `curl`.
- [Uptime monitors](../docs/monitors.md) — monitor fields and notification channels.
