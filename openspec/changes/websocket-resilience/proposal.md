## Why

Only three of the seven WebSocket clients reconnect (`controller.ts:80`, `index.ts:478`, `startlights.ts:100`); the controller retries at a fixed 5s with no backoff or jitter, while `tv.ts`, `spectator.ts`, `pitboard.ts`, and `player.ts` never reconnect at all. Broadcasts carry no sequence number, so clients cannot tell they missed a message, and a reconnecting client has no state catch-up — stale screens persist silently until a full page reload.

## What Changes

- **Shared reconnect helper** (`ts/ws.ts`) with exponential backoff + full jitter, a maximum delay, `onerror` handling, and backoff reset on successful open; adopted by every WS page.
- **Sequence numbers**: every server broadcast carries a global monotonic `seq`; clients track the last seen value and detect gaps.
- **Resync**: on gap detection (or reconnect), clients send `{type:'resync', topics:[...]}` and receive a fresh snapshot.
- **`hello` snapshot**: immediately after connect the server sends the current state for the connection's topics, so a fresh or reconnecting client is correct without waiting.
- **Idempotent state application**: clients replace snapshot-backed state (racers, standings, flags, weather) and de-duplicate append-only streams (commentary, race radio) by id.
- **BREAKING (internal protocol)**: the raw `[]Racer` broadcast becomes a `{type:'racers', seq, payload}` envelope so it can carry `seq`; all in-repo clients are updated in this change.
- **Tests**: backoff scheduling, gap detection → resync, snapshot on connect, snapshot idempotency.

## Capabilities

### New Capabilities
- `websocket-resilience`: automatic reconnection with backoff, sequenced broadcasts with gap detection, connect-time `hello` snapshots, and resync that restores client state.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `ws/ws.go`, `app/app.go`: global sequence counter; snapshot builder; `hello` on connect; `resync` inbound handler; wrap the racer broadcast.
- New `ts/ws.ts` shared reconnect/subscribe/seq helper; adopted by all page scripts.
- Protocol break for the racer array — all `ts/*.ts` `onmessage` handlers updated in the same change.
- Tests: Go WS tests + Playwright reconnect coverage.

## Dependencies

- Builds on `websocket-hardening` (connection writer) and `websocket-auth-rooms` (topics/identity for per-connection snapshots).
- The `hello`/resync snapshot includes `race_state` when `live-race-state-broadcast` is present; otherwise it covers flags, weather, racers, and standings.
