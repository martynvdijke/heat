## Context

Race state is client-local in `ts/controller.ts` (`raceState`, `raceSeconds`, `currentLap`): refreshing the controller loses the running race, and TV, spectator, and pitboard have no synchronized clock, lap, or standings. Gap computation lives only in the controller (`computeGaps`, `ts/controller.ts:148`), and the server already broadcasts `race_radio` (`ws/ws.go:158-170`, produced by `handlers/race_radio.go:84`) that no client consumes. The `websocket-auth-rooms` change adds topic subscriptions; `websocket-resilience` adds snapshot recovery on reconnect.

Stakeholders: race director (controller), TV/spectator/pitboard viewers, driver/pitboard consumers of race radio.

## Goals / Non-Goals

**Goals:**
- Server-authoritative race state (`stopped|racing|paused`) with elapsed time, current lap, and total laps as single source of truth.
- Persistence across server restart and page reload.
- `race_state` broadcast on transitions and 1s tick while racing; `standings` broadcast on lap/position changes.
- Synchronized rendering of clock, lap counter, and standings on TV, spectator, pitboard, and controller (no divergent client timers).
- Race radio delivery to controller, driver, and pitboard (server already broadcasts).

**Non-Goals:**
- Per-sector live telemetry streaming.
- Full race simulation / history playback.
- Template management or i18n for race radio.
- Audio/voice narration.

## Decisions

### D1: Persistence via single-row `race_state` table
Schema `race_state(id INTEGER PRIMARY KEY CHECK (id=1), state TEXT NOT NULL, started_at TEXT NULL, accumulated_ms INTEGER NOT NULL DEFAULT 0, current_lap INTEGER NOT NULL DEFAULT 0, total_laps INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL)` added as a startup migration in `db/init.go` (same append-to-sequence style as existing migrations). Single row with `id=1` enforces singleton. Rejected: in-memory only — fails reload/restart survival, which is the primary motivation.

### D2: Elapsed computed server-side
While `racing`, elapsed is `accumulated_ms + (now - started_at)`; on pause, elapsed is folded into `accumulated_ms` and `started_at` nulled; on stop, both reset and lap reset. Clients never compute elapsed locally from a start timestamp; they render `elapsed_ms` from broadcasts. Alternative of sending `started_at` and letting clients tick locally was rejected — would reintroduce divergent clocks.

### D3: Dedicated race-state endpoints
`POST /api/race/state` with body `{action: "start"|"pause"|"resume"|"stop", total_laps?: number}` and `GET /api/race/state` returning `{state, elapsed_ms, current_lap, total_laps}`. POST requires authenticated controller role, session auth, and CSRF (same middleware as other controller endpoints). Immediate `race_state` broadcast on success. Rejected: reusing `POST /api/flags` — different semantics (flag state vs race lifecycle), different persistence, and would conflate authorization concerns.

### D4: Standings computed in Go mirroring `computeGaps`
Go implementation mirrors `ts/controller.ts:148` `computeGaps` semantics: leader gap is `LEAD`, lapped racers show `+N` (laps down), same-lap racers show empty/lap-gap per existing logic. Broadcast as `standings` event-driven on lap recorded (`POST /api/lap-records` or equivalent) and on position change, NOT on the 1s tick. Polling `/api/lap-records` remains as fallback. Rejected: computing gaps on clients — duplicates logic, risks divergence, and TV/spectator would need lap-record polling.

### D5: 1s tick only while racing
A `time.Ticker` runs only in `racing` state; it is stopped on pause/stop and restarted on resume/start. Each tick broadcasts `race_state` with current `elapsed_ms`. Overhead is one broadcast per second to subscribed clients — trivial. Rejected: ticking in all states — wasteful and would mask the semantic difference between stopped/paused/racing.

### D6: Race radio — add TS receiver only
Server already broadcasts `race_radio` via `ws/ws.go:158-170` from `handlers/race_radio.go:84` wrapped as `{type:'race_radio', ...}`. This change adds only the TypeScript receiver/handler in `ts/controller.ts`, `ts/driver.ts`, `ts/pitboard.ts` (and shared helper if present) to surface messages in the UI. No new Go broadcast path needed.

### D7: WS topic subscription with fallback
Views subscribe to `race_state`/`standings` (and `race_radio` where relevant) via `websocket-auth-rooms` topics when that change is present; if absent, the server delivers those broadcasts unconditionally to all connected clients (same pattern as existing `BroadcastRaceRadio`). Clients also handle snapshot recovery from `websocket-resilience` on reconnect.

## Risks / Trade-offs

- [Multiple controllers issuing concurrent transitions; last-write-wins] → Server serializes state transitions under a mutex / single-writer channel; invalid transitions (e.g., pause while stopped) return 409.
- [1s tick fan-out overhead] → One message per second per deployment; payload is small JSON; trivial for expected client counts (<100). If scale grows, add per-room filtering.
- [Migration on existing deployments] → Single-row table creation is idempotent (`CREATE TABLE IF NOT EXISTS` + `INSERT OR IGNORE` seed row with `stopped`); no data to migrate; rollback is dropping the table and removing routes.
- [Clock drift between server `now` and DB timestamps] → Elapsed derived from monotonic server `now` minus `started_at` (stored as UTC ISO string); `accumulated_ms` is authoritative; DB `updated_at` is informational only.
- [Standing computation divergence from controller] → Go implementation is tested against controller `computeGaps` fixtures to ensure identical `LEAD`/`+N`/empty outputs.

## Migration Plan

1. Deploy migration: `db/init.go` creates `race_state` table and seeds `id=1` as `stopped` if absent.
2. Deploy new endpoints and broadcaster goroutine; register routes in `main.go`; wire `RaceStateBroadcast` channel in `app/app.go`.
3. Deploy TS view updates; controller stops local timer in favor of WS state.
4. Rollback: revert binary; `race_state` table is inert without code reading it; drop if desired.
5. No feature flag required; change is additive and backward-compatible (existing lap-record polling continues as fallback).

## Open Questions

- None blocking; `total_laps` source of truth when controller starts a race (POST body vs race config) to be aligned with existing race creation flow.
