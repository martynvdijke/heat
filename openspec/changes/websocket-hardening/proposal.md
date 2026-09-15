## Why

The WebSocket hub (`ws/ws.go`) has no liveness detection, no message-size limit, and serializes every broadcast behind a single global mutex (`wsWriteMu`) held across blocking network writes to every client. A single slow or half-open client stalls broadcasts for all viewers, dead peers are never reaped, and the unbuffered broadcast channels silently drop messages under load with no signal.

## What Changes

- **Heartbeat**: the server pings each connection on an interval, sets a read deadline, and closes connections that miss a pong (`ws/ws.go` currently has no `SetReadDeadline` / `SetPongHandler`).
- **Read limit**: `SetReadLimit` on upgrade; oversized frames close the connection.
- **Per-connection writer**: each client gets a buffered outbound queue and a dedicated writer goroutine with write deadlines. The global `wsWriteMu` held across `WriteJSON` is removed; a slow client no longer blocks others.
- **Lagging-client eviction**: a client whose outbound queue stays full is disconnected and cleaned up rather than degrading the hub.
- **Buffered broadcast channels** with an explicit drop policy and a `heat_ws_broadcast_dropped_total` counter so drops are observable instead of silent.
- **Tests**: heartbeat reaping, read-limit rejection, slow-client isolation, drop counter.

## Capabilities

### New Capabilities
- `websocket-hardening`: connection liveness (heartbeat + read deadline), inbound message-size limits, per-connection write isolation, lagging-client eviction, and observable broadcast backpressure.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `ws/ws.go`: heartbeat, read limit, per-connection writer, eviction, drop counter (primary change).
- `app/app.go`: buffered server channels; connection wrapper type with outbound queue.
- `main.go`: broadcast goroutine startup unchanged; `/ws` route unchanged.
- `telemetry.go`: new dropped-broadcast counter metric.
- Tests: Go WebSocket tests (extend `09_test_infrastructure_test.go` or add `19_test_ws_test.go`).
