## 1. Core hardening

- [ ] 1.1 Set read limit 64 KiB via `SetReadLimit` on upgrade in `ws/ws.go`; close connections with oversized frames
- [ ] 1.2 Add heartbeat: periodic ping, `SetReadDeadline` + `SetPongHandler`, close+remove connections that miss a pong
- [ ] 1.3 Introduce connection wrapper type with buffered `send` channel (~256) and per-connection writer goroutine with `SetWriteDeadline`; only the writer writes to the conn (including pings)
- [ ] 1.4 Rewrite `broadcastToClients` to marshal once and enqueue non-blocking to each client's `send`; lagging-client eviction when queue full (disconnect and cleanup, never block the producer)
- [ ] 1.5 Remove global `wsWriteMu` held across `WriteJSON` (ws/ws.go:84-106)

## 2. Backpressure observability

- [ ] 2.1 Buffer server broadcast channels (~64) and apply drop-newest non-blocking policy when full
- [ ] 2.2 Add `heat_ws_broadcast_dropped_total` counter in `telemetry.go` and increment on each dropped broadcast plus emit a log line

## 3. Tests and validation

- [ ] 3.1 Go tests: dead client reaped by heartbeat after missing pong
- [ ] 3.2 Go tests: oversized message (>64 KiB) closes the connection
- [ ] 3.3 Go tests: slow client does not block delivery to healthy clients (per-connection write isolation)
- [ ] 3.4 Go tests: broadcast drop increments `heat_ws_broadcast_dropped_total` and log is emitted; full outbound queue evicts only the lagging client
- [ ] 3.5 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
