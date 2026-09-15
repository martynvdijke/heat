## 1. Shared WebSocket helper

- [ ] 1.1 Create `ts/ws.ts` with `connectWithRetry(url, {topics, onMessage, onOpen})` — exponential backoff (base ~500ms, factor 2, cap ~30s), full jitter, `onerror` treated as disconnect, reset on `onopen`, `lastSeq` tracking, gap detection and `resync` dispatch
- [ ] 1.2 Adopt `ts/ws.ts` in `controller.ts`, `index.ts`, `startlights.ts` — replace fixed 5s retry with the shared helper
- [ ] 1.3 Adopt `ts/ws.ts` in `tv.ts`, `spectator.ts`, `pitboard.ts`, `player.ts` — add reconnect where none exists today

## 2. Server: sequence numbers and envelope

- [ ] 2.1 Add global atomic sequence counter and wrap every outbound broadcast in an envelope carrying `seq` (`{type:'racers', seq, payload}` for racers; `{type, topic, seq, payload}` for other typed messages — note current raw `[]Racer` and `ws.go` map shapes)
- [ ] 2.2 Update all `ts/*.ts` `onmessage` handlers to unwrap the racer envelope (`payload` field) and read `seq` from envelopes

## 3. Server: snapshot and resync

- [ ] 3.1 Implement `buildSnapshot(identity, topics)` — collects flags, weather, racers, standings, and `race_state` when `live-race-state-broadcast` is present, scoped by `websocket-auth-rooms` topics/identity (state the dependency)
- [ ] 3.2 Send `hello` snapshot on every successful WebSocket connect (via `buildSnapshot` scoped to the connection's topics)
- [ ] 3.3 Handle inbound `{type:'resync', topics:[...]}` — reply with a fresh topic-scoped snapshot via `buildSnapshot`

## 4. Client: gap detection and idempotent snapshot application

- [ ] 4.1 Implement gap detection on `seq` jump or reconnect — send `resync` for subscribed topics
- [ ] 4.2 Implement idempotent snapshot application — replace keyed collections (racers, standings, flags, weather, race_state), de-duplicate append streams (commentary, race radio) by `id`

## 5. Tests and validation

- [ ] 5.1 Tests: backoff scheduling (jitter, cap, reset on open), gap detection triggers resync, `hello` snapshot on connect, snapshot idempotency (double-apply produces no duplicates)
- [ ] 5.2 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
