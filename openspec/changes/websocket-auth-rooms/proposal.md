## Why

`GET /ws` (`main.go:331`) is unauthenticated, yet any connected client can inject `flag`, `self_service`, `lap_update`, and `weather_update` broadcasts for every viewer (`ws.go:57-79`), and `BroadcastSelfService` sends every player's gear/stress/turbo telemetry to all clients (`ws.go:189-200`) with `player.ts` filtering only on the client. There is no server-side notion of who a client is or what it may send or receive, so integrity and privacy depend entirely on well-behaved clients.

## What Changes

- **Handshake classification**: the upgrade derives the connection identity from the session cookie or a player token; invalid/expired credentials are rejected, while absent credentials yield a read-only spectator identity (so public TV/spectator displays keep working). Unauthenticated clients must not receive sensitive topics or emit mutations.
- **Connection identity**: each connection carries a role (`controller` | `player` | `spectator` | `tv` | `pitboard`) and optional `racer_id`.
- **Topic subscriptions**: clients send `{type:'subscribe', topics:[...]}`; broadcasts are tagged with a topic and delivered only to subscribers. Defaults preserve today's public read behavior.
- **Inbound authorization**: only an authenticated controller/admin may emit `flag`, `lap_update`, and `weather_update`; a player may only emit `self_service` for its own racer id.
- **Targeted notifications**: `{type:'notify', racer_id, ...}` delivered only to that racer's connections, with an optional client ack.
- **Presence**: the server tracks connected clients by role/racer/last-seen and broadcasts presence changes; the controller shows connected roles.
- **Tests**: invalid-credential rejection, topic filtering, spoofed-racer rejection, targeted delivery, presence.

## Capabilities

### New Capabilities
- `websocket-auth-rooms`: authenticated/classified WebSocket handshake, per-connection identity and roles, topic subscriptions with filtered delivery, inbound authorization, targeted driver notifications, and presence.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `ws/ws.go`, `app/app.go`, `main.go`: auth/identity plumbing on `/ws`; connection registry becomes metadata-bearing.
- Reuse existing `middleware` session validation and player-token validation.
- All TS clients: supply credentials, subscribe to topics, handle `notify` (`controller.ts`, `tv.ts`, `spectator.ts`, `pitboard.ts`, `player.ts`, `index.ts`).
- Tests: Go handler/WS tests + Playwright.

## Dependencies

- Builds on `websocket-hardening` (connection wrapper with outbound queue); the registry metadata added here is consumed by `websocket-resilience` (snapshots) and `live-race-state-broadcast` (subscriptions).
