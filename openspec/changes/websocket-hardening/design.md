## Context

`ws/ws.go` is the sole WebSocket hub. Upgrade registers the conn in a global map protected by a mutex, but all subsequent `WriteJSON` calls in `broadcastToClients` are serialized behind a second global mutex `wsWriteMu` held across blocking network writes (lines 84-106). `app/app.go` and `handlers/flags.go` feed the hub via unbuffered channels with `select { case ch <- msg: default: }` drops (handlers/flags.go:30-33, ws/ws.go:220-223) that are silent today. There is no `SetReadLimit`, no `SetReadDeadline`/`SetPongHandler`, and no heartbeat — half-open peers are never reaped. A single stalled client stalls every broadcast; an oversized frame can exhaust memory; dropped broadcasts are invisible.

Stakeholders: TV/spectator/controller clients over `/ws`; `ws` package, `app.Server`, `telemetry.go`, broadcast goroutines.

## Goals / Non-Goals

**Goals:**
- Detect and reap dead/half-open peers via periodic pings + read deadlines.
- Bound inbound frame size.
- Isolate per-connection writes so one slow client does not block others.
- Evict lagging clients whose queue is full instead of blocking the hub.
- Make broadcast backpressure observable (buffered channels, drop-newest with counter + log).

**Non-Goals:**
- Topics / pub-sub routing or auth — separate change.
- Sequence numbers, replay, or snapshots — separate change.
- Protocol / message-type changes — separate change.
- Template or UI changes unrelated to the hardening.

## Decisions

### D1: Heartbeat via gorilla control frames (`SetReadDeadline` + `SetPongHandler`)

Server pings every connection on a fixed interval; on each ping it sets a read deadline, installs a `SetPongHandler` that refreshes the deadline on pong, and closes+removes connections that miss a pong. Only the per-connection writer goroutine writes pings (gorilla forbids concurrent writes).

*Rejected:* App-level JSON heartbeat — reinvents the WebSocket control-frame protocol, adds payload parsing, and duplicates what gorilla already implements.

### D2: Read limit 64 KiB via `SetReadLimit`

Set on upgrade immediately after `Upgrader.Upgrade`. Oversized frames cause the read to fail and the connection to be closed.

### D3: Per-connection writer goroutine + buffered `send` channel; single-writer invariant

Each connection is wrapped as `{ conn *websocket.Conn, send chan []byte, done chan struct{} }` with a dedicated writer goroutine that is the sole writer to the conn (including pings, satisfying the gorilla concurrency constraint). `broadcastToClients` marshals the payload once and enqueues the bytes to each client's `send` with a non-blocking send (see D4). This removes the global `wsWriteMu` currently held across `WriteJSON` (ws/ws.go:84-106). Writes use `SetWriteDeadline`.

*Interim alternative considered and rejected:* Keep `wsWriteMu` and just add a `SetWriteDeadline` around `WriteJSON`. Rejected because it still serializes all writes — a slow client still blocks every other client for the duration of the deadline.

### D4: Queue capacity ~256; full → evict the client

If `send` is full, the producer never blocks — instead the lagging client is disconnected and cleaned up. Eviction (not drop-for-slow-client) is chosen so a broken client cannot silently miss state; it reconnects via the existing resilience path (see `websocket-resilience`).

### D5: Server broadcast channels buffered (~64), drop-newest with counter

Current unbuffered `select default` at `handlers/flags.go:30-33` and `ws/ws.go:220-223` is the behavior being made observable. Channels become buffered (~64); when full the newest message is dropped non-blocking and `heat_ws_broadcast_dropped_total` increments plus a log line is emitted.

### D6: Only the writer goroutine writes to the conn

No other goroutine writes directly — reads run on the read loop goroutine; all writes (messages + pings) are funneled through the writer's `send` channel or its ticker. This is required because gorilla forbids concurrent `Write*` calls.

## Risks / Trade-offs

- [Ping interval too aggressive reaps healthy clients on transient latency] → Choose conservative default (e.g., 30s ping / 60s deadline) and make it a const for tuning.
- [Evicting lagging clients causes reconnect churn] → Queue depth 256 and buffered hub channels reduce spurious evictions; reconnect is already handled by `websocket-resilience`.
- [Per-connection goroutine increases goroutine count] → One goroutine per viewer is trivial at expected fan-out (<1000); bounded `send` prevents unbounded memory.
- [Dropping newest broadcast may lose the latest state] → Counter + log make it observable; clients catch up via polling/snapshot path from the companion change.
- [Removing `wsWriteMu` changes concurrency model] → Single-writer per conn eliminates the need; hub map access remains mutex-protected.

## Migration Plan

1. Add connection wrapper type and per-connection writer in `app/app.go` / `ws/ws.go`; buffer server channels; add `heat_ws_broadcast_dropped_total` in `telemetry.go`.
2. Switch `broadcastToClients` to marshal-once + non-blocking enqueue; remove `wsWriteMu`; add `SetReadLimit` and ping/pong deadline logic.
3. Add Go tests (heartbeat reaping, oversize rejection, slow-client isolation, drop counter) and run `task pre-push`.
4. Deploy — no schema or wire-protocol changes; rollout is backward compatible. Rollback: revert the change; old unbuffered drop behavior resumes (with silent drops).

## Open Questions

- Exact ping interval and read-deadline values (default proposed: 30s / 60s) — confirm against expected viewer latency.
- Final queue/channel capacities (256 / 64) — confirm under load test; constants are easy to tune after.
