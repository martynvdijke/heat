## Why

Race state is client-local in `ts/controller.ts` (`raceState`, `raceSeconds`, `currentLap`): refreshing the controller loses the running race, and TV, spectator, and pitboard have no synchronized clock, lap, or standings at all. Gap computation lives only in the controller (`computeGaps`, `ts/controller.ts:148`), and the server already broadcasts `race_radio` (`ws.go:158-170`, produced by `handlers/race_radio.go:84`) that no client consumes.

## What Changes

- **Server-authoritative race state**: `stopped | racing | paused` with elapsed time, current lap, and total laps; transitions are accepted only from an authenticated controller and persisted so they survive reload/restart.
- **`race_state` broadcast**: sent on every transition and on a 1s tick while racing.
- **Server-computed standings/gaps**: positions and lap gaps computed server-side from racers + lap records and broadcast as `standings` on lap/position changes, mirroring the controller's current `computeGaps` semantics.
- **Synchronized views**: TV, spectator, pitboard, and controller render the shared clock, lap counter, and standings.
- **Race radio receiver**: `type:'race_radio'` consumed on the controller, driver, and pitboard pages (server side already broadcasts it).
- **Controller simplification**: the controller consumes WS-pushed standings and race state; polling `/api/lap-records` remains only as a fallback.
- **Tests**: state transitions, persistence across reload, standings broadcast, radio delivery.

## Capabilities

### New Capabilities
- `live-race-state-broadcast`: authoritative persisted race state, broadcast state ticking, server-computed live standings, synchronized multi-view rendering of clock/lap/standings, and race-radio delivery to clients.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `db/init.go`: `race_state` single-row table migration.
- New `handlers/race_state.go` (or extension of `handlers/race.go`): state endpoints + broadcaster; server channel in `app/app.go`.
- `ws/ws.go`: `race_state` / `standings` broadcasts; race-radio already broadcast.
- `main.go`: routes (authenticated + CSRF).
- `ts/controller.ts`, `ts/tv.ts`, `ts/spectator.ts`, `ts/pitboard.ts`, `ts/player.ts`, `ts/driver.ts`: render shared state.
- Tests: Go + Playwright.

## Dependencies

- Uses the topic subscriptions from `websocket-auth-rooms` (falls back to unconditional delivery if that change is not present) and benefits from `websocket-resilience` snapshots so a reloaded view repaints instantly.
