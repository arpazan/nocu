# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Primary users are on-shift NOC operators: a small operations team watching a wall or desk screen during a shift, reacting to monitor status flips and triaging incidents as they happen. Their job is to notice a change fast, decide whether it needs a human, and move the incident through its lifecycle.

Three roles exist and the UI must respect them:

- **viewer**: the default for new accounts. Reads incidents and monitors and receives realtime events. Cannot create or change anything.
- **operator**: creates and updates incidents and monitors, pauses and resumes probes.
- **admin**: operator rights plus administration (role grants are done directly in the database today; no admin UI exists yet).

No other audience is confirmed. Do not design for customers, executives, or a public status page unless that is added here.

## Product Purpose

Nocu is a self-hosted NOC platform. It probes HTTP(S) endpoints on a fixed interval, records a heartbeat per probe, raises `monitor.up` / `monitor.down` when a status flips, warns about expiring TLS certificates and domains, and tracks human-managed incidents with severity and a lifecycle. Everything the backend emits is pushed to the browser over one WebSocket so the operator never has to refresh.

Success for an operator: the current situation is understood within seconds of looking at the screen, anything that needs attention is impossible to miss, and acting on it (acknowledge, resolve, pause, edit) takes one step from where it is seen.

## Positioning

Incidents and uptime in one realtime view. Pure uptime checkers show monitors; ticketing tools show incidents. Nocu puts monitor flips and human-tracked incidents in the same feed and the same situation picture, so the operator reads one surface instead of two.

## Operating Context

- Operators sit on a shift and keep the app open for hours, often on a large shared screen, so the overview is a persistent display, not a page visited briefly.
- The realtime connection can drop; the UI must show whether it is live or reconnecting, and events arrive continuously while the operator is looking elsewhere.
- Monitors probe public URLs only by default; loopback and private targets are refused unless the scheduler explicitly allows them.
- Notification channels are attached per monitor: Discord webhooks and generic webhooks are delivered today; browser web push is stored but not yet delivered (needs VAPID keys).
- Role grants happen outside the UI for now, so a viewer who needs operator rights sees controls disabled with the reason "Needs the operator role".
- Development runs the four Go services, PostgreSQL, and NATS alongside the Next.js dev server on port 3000.

## Capabilities and Constraints

Routes: Overview (`/`), Monitors (list, detail, new, edit), Incidents (list, detail), plus login and register.

Terminology (from the data model; use these words in the UI):

- **Monitor**: one HTTP(S) URL probed on an interval, with retries, timeout, method, body, headers, auth, redirect limit. Can be paused and resumed. Has a last status and a heartbeat history.
- **Heartbeat**: one probe result.
- **Incident**: severity `critical`, `high`, `medium`, or `low`; status `open` → `acknowledged` → `resolved` → `closed`. Closing is allowed from any non-closed state; a resolved incident can reopen. A `source` string records where it came from.
- **Event**: anything the backend publishes (`incident.*`, `job.*`, `monitor.*`), persisted and pushed live.

Constraints:

- Secrets on a monitor (basic password, bearer token) are write-only and never returned by the API; forms must not pretend to show them.
- The web app is a Next.js App Router application on Tailwind v4 with shadcn/ui on Radix; new UI is built from the existing `src/components/ui` set rather than a new kit.
- English only. No internationalisation is planned.
- Realtime data comes from a WebSocket; the overview keeps a bounded in-memory event feed.
- Mobile web must work (the shell already collapses to a top bar and sheet), but the primary scene is a wide screen.

Undecided:

- Whether a public status page or customer-facing surface will ever exist.
- Whether role administration moves into the UI.
- Web push delivery timing.

## Brand Commitments

- The product name is **Nocu**. The wordmark is the name set next to the lucide `Activity` icon; this mark is binding.
- The current typefaces (Manrope for UI, Geist Mono for data) and the orange primary are the incumbent system but were not pinned as binding; future work may revisit them with the user's agreement, and must document the result in DESIGN.md.
- Voice in the UI is short, factual, and operator-facing: states are named plainly ("Live", "Reconnecting…", "Nothing needs attention right now").

## Evidence on Hand

- Real product copy and states live in the components under `src/components`; no marketing copy exists.
- No logo file exists beyond the lucide icon; `public/` holds only Next.js starter SVGs.
- No testimonials, customers, benchmarks, pricing, or screenshots exist. Do not fabricate any.
- Demo data is created through the API (see the root README); there is no seeded sample dataset in the web app.

## Product Principles

1. **Situation first.** The first thing on screen answers "is anything wrong right now?" before any navigation or detail.
2. **Attention is the scarce resource.** Status changes must be unmissable; everything that is fine should recede.
3. **One step from seeing to acting.** Acknowledge, resolve, pause, and edit are reachable from where the item is shown, and are visibly gated by role rather than hidden.
4. **Trust the feed.** Live state is always labelled, time is always relative and current, and stale or disconnected states are stated, never implied.
5. **Consistency over expression.** This is an Operate surface used for hours at a time; patterns the operator already knows beat novelty.

## Accessibility & Inclusion

No formal standard has been mandated. Established practice in the codebase: keyboard-visible focus rings on every interactive element, `aria-live` on connection state, `aria-current` on navigation, motion respects `prefers-reduced-motion`. Future work keeps these at minimum.
