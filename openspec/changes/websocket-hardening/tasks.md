## 1. Core hardening

- [x] 1.1 Set read limit 64 KiB via `SetReadLimit` on upgrade in `ws/ws.go`; close connections with oversized frames
- [x] 1.2 Add heartbeat: periodic ping, `SetReadDeadline` + `SetPongHandler`, close+remove connections that miss a pong
- [x] 1.3 Introduce connection wrapper type with buffered `send` channel (~256) and per-connection writer goroutine with `SetWriteDeadline`; only the writer writes to the conn (including pings)
- [x] 1.4 Rewrite `broadcastToClients` to marshal once and enqueue non-blocking to each client's `send`; lagging-client eviction when queue full (disconnect and cleanup, never block the producer)
- [x] 1.5 Remove global `wsWriteMu` held across `WriteJSON` (ws/ws.go:84-106)

## 2. Backpressure observability

- [x] 2.1 Buffer server broadcast channels (~64) and apply drop-newest non-blocking policy when full
- [x] 2.2 Add `heat_ws_broadcast_dropped_total` counter in `telemetry.go` and increment on each dropped broadcast plus emit a log line

## 3. Tests and validation

- [x] 3.1 Go tests: dead client reaped by heartbeat after missing pong
- [x] 3.2 Go tests: oversized message (>64 KiB) closes the connection
- [x] 3.3 Go tests: slow client does not block delivery to healthy clients (per-connection write isolation)
- [x] 3.4 Go tests: broadcast drop increments `heat_ws_broadcast_dropped_total` and log is emitted; full outbound queue evicts only the lagging client
- [x] 3.5 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
