## 1. Handshake classification and identity

- [x] 1.1 Classify `GET /ws` upgrade: valid session cookie → `controller`; valid player token via `Sec-WebSocket-Protocol` → `player` bound to `racer_id`; no credentials → `spectator`; invalid/expired → reject 401
- [x] 1.2 Define connection metadata struct (`role`, `racer_id`, `connectionId`, `lastSeen`, `topics`, `writer`) and wire it into the WS registry
- [x] 1.3 Coordinate registry change (`map[*websocket.Conn]bool` → `map[*websocket.Conn]*ConnMeta`) with `websocket-hardening` writer wrapper

## 2. Topic subscriptions and filtered delivery

- [x] 2.1 Implement topic registry and role→default topics map (public: `flags`, `racers`, `commentary`, `weather`, `race_state`; restricted: `telemetry`, `presence`, `race_radio`)
- [x] 2.2 Handle `{type:'subscribe', topics:[...]}` to replace per-connection subscription set; tag every outbound envelope with `topic`
- [x] 2.3 Refactor `broadcastToClients` to deliver only to subscribers of the message topic; preserve public broadcast semantics for default subscribers

## 3. Inbound authorization

- [x] 3.1 Authorize `flag`/`lap_update`/`weather_update` to `controller` role only; reject and do not re-broadcast on violation
- [x] 3.2 Authorize `self_service` to `player` only when `racer_id` matches bound `racer_id`; reject spoofed racer_id
- [x] 3.3 Ensure unauthorized inbound messages return an error/close frame and are not broadcast

## 4. Targeted notifications and presence

- [x] 4.1 Implement `{type:'notify', racer_id, ...}` targeted delivery filtering registry by `racer_id`
- [x] 4.2 Handle `{type:'notify_ack', id}` from client and relay/record ack
- [x] 4.3 Track presence by role/racer/last-seen; broadcast presence changes to `presence` subscribers; controller renders connected roles

## 5. TypeScript client updates

- [x] 5.1 Update all TS clients (`controller.ts`, `tv.ts`, `spectator.ts`, `pitboard.ts`, `player.ts`, `index.ts`) to supply credentials (cookie auto; player token via `Sec-WebSocket-Protocol`) and send `subscribe` on open
- [x] 5.2 Handle `notify` display and send `notify_ack` where applicable; handle `topic` field on inbound envelopes

## 6. Tests and validation

- [x] 6.1 Go tests: handshake rejection for invalid/expired credentials; spectator allowed with no credentials
- [x] 6.2 Go tests: topic filtering — subscribed receives, unsubscribed does not; defaults preserve TV public behavior
- [x] 6.3 Go tests: spoof rejection — player cannot emit `self_service` for another racer; spectator cannot emit `flag`
- [x] 6.4 Go tests: targeted delivery — only target racer's connections receive `notify`; `notify_ack` round-trip
- [x] 6.5 Go tests: presence — connect/disconnect emits update to `presence` subscribers, not to non-subscribers
- [x] 6.6 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
