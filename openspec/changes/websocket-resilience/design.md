## Context

Only three of seven WebSocket pages reconnect today (`controller.ts:80`, `index.ts:478`, `startlights.ts:100`) with a fixed 5s delay, no jitter, and no `onerror` handling; `tv.ts`, `spectator.ts`, `pitboard.ts`, and `player.ts` never reconnect. Broadcasts carry no sequence number, so silent message loss is undetectable, and reconnecting clients have no catch-up path — stale flags, weather, standings, and racer positions persist until a full reload.

Existing building blocks:
- `ws/ws.go`: inbound switch (`flag|self_service|lap_update|weather_update`), per-`Broadcast*` goroutines fed by server channels; racers are broadcast as a raw `[]Racer` JSON array, other topics as `{type, ...}` maps assembled in `ws.go`.
- `app/app.go` and `ws/ws.go`: server state for flags, weather, racers, standings, and (when `live-race-state-broadcast` is present) `race_state`.
- Prior changes `websocket-hardening` (connection writer / lifecycle) and `websocket-auth-rooms` (per-connection `topics` and `identity` used to scope snapshots) are dependencies — this change reuses their topics/identity plumbing.
- All in-repo WS clients live in `ts/*.ts` (controller, tv, spectator, pitboard, player, index, startlights).

## Goals / Non-Goals

**Goals:**
- Every WS page reconnects automatically with bounded, jittered backoff and recovers current state without a reload.
- Clients can detect missed messages via sequence numbers and resync on demand.
- A fresh or reconnected client is correct immediately via a server-pushed `hello` snapshot scoped to its topics.
- Internal protocol carries `seq` on every broadcast (requires wrapping the raw racer array).

**Non-Goals:**
- Offline queueing / guaranteed delivery — the system detects gaps and offers snapshot resync, it does not queue and replay every missed message. Clients that were offline for an extended period converge via snapshot, not a replay log.
- Cross-tab coordination or Service Worker persistence.
- Auth or room semantics — owned by `websocket-auth-rooms`; this change only consumes `topics`/`identity`.

## Decisions

### D1: Shared `ts/ws.ts` helper `connectWithRetry(url, {topics, onMessage, onOpen})`
All pages use one helper that owns `WebSocket` lifecycle, `onopen`/`onclose`/`onerror`/`onmessage`, backoff timers, jitter, and `seq` gap detection + `resync` dispatch. Rejected: per-page hand-rolled retry (the status quo that left four pages without reconnect and duplicated the fixed-delay logic).

### D2: Global atomic sequence counter server-side
A single `atomic.Uint64` (or `sync/atomic` counter) incremented for every outbound broadcast, injected as `seq` into each envelope. Rejected: per-topic sequence numbers — the topic set is dynamic (depends on `websocket-auth-rooms` subscriptions) and a single monotonic counter is sufficient to detect any loss; per-topic counters add complexity without benefit.

### D3: Envelope shape
- Typed messages: `{type, topic, seq, payload}`.
- Racers (formerly raw `[]Racer`): `{type:'racers', seq, payload: []Racer}`.
This is a BREAKING internal protocol change — the raw array cannot carry `seq`. All in-repo `onmessage` handlers are updated in the same change. Current shapes to note: raw `[]Racer` array for racers; maps built in `ws.go` for flags/weather/standings/race_state/race_radio/commentary wrapping with `type`.

### D4: Server `buildSnapshot(identity, topics)` reused by `hello` and `resync`
Single snapshot builder that reads current flags, weather, racers, standings, and `race_state` (when `live-race-state-broadcast` is present) and filters/scopes by the connection's `topics`/`identity` from `websocket-auth-rooms`. Both the connect-time `hello` (`{type:'hello', seq, snapshot}`) and the inbound `resync` reply (`{type:'resync', seq, snapshot}` scoped to `topics` in the request) call it. Dependency on `websocket-auth-rooms` is explicit — without topics/identity the snapshot cannot be scoped.

### D5: Backoff — base ~500ms, factor 2, cap ~30s, full jitter, reset on open
`delay = min(cap, base * 2^attempt)` then `actual = random(0, delay)` (full jitter). `onerror` is treated like `onclose` for scheduling. Attempt counter resets to 0 on `onopen`. Rationale: fast first retry for transient blips, bounded worst-case, jitter avoids reconnect stampedes after a server restart.

### D6: Snapshot idempotency
- Snapshot-backed collections (racers, standings, flags, weather, race_state): replace the entire keyed collection with `snapshot` contents.
- Append-only streams (commentary, race radio): merge by stable `id` — existing ids are kept, new ids appended, duplicates ignored. Applying the same snapshot twice is a no-op for streams and a stable replace for collections. Client keeps `lastSeq` and updates it to `max(lastSeq, envelope.seq)` after applying.

## Risks / Trade-offs

- [Protocol break — racers envelope] → All in-repo clients updated atomically in this change; no backwards-compat shim is kept (internal protocol only). External consumers, if any, must update.
- [Reconnect stampede after server restart] → Full jitter on every retry spreads reconnects; cap prevents unbounded growth.
- [Snapshot cost on every connect + resync] → Snapshot reads in-memory server state only (no DB round-trip except where state itself is DB-backed and already cached); payload is bounded by current race size. If races grow large, paginate or compress (not in this change).
- [Global seq wraps / resets on server restart] → Counter resets on restart; clients treat a reconnect (which always follows a restart) as a resync point and accept the post-restart `seq` as new baseline after applying `hello`.
- [Dependency on websocket-auth-rooms topics] → If that change is not present, snapshots fall back to unscoped (all topics); noted as a dependency.

## Migration Plan

1. Land `websocket-hardening` and `websocket-auth-rooms` first (dependencies).
2. Land this change atomically: server counter + envelope wrapping + `buildSnapshot` + `hello` + `resync` handler together with `ts/ws.ts` and all `onmessage` updates — no partial rollout where some clients expect the raw array.
3. Deploy; verify WS reconnect in staging by killing the server and observing jittered reconnect + `hello` snapshot.
4. Rollback: revert the single commit — clients and server revert together; no data migration to undo.

## Open Questions

- None — decisions above cover the scope. Snapshot payload shape for `resync` (full vs. topic-filtered) is topic-filtered as described in D4.
