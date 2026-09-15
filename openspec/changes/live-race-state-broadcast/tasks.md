## 1. Backend: race_state persistence and model

- [x] 1.1 Add `race_state` single-row table migration to `db/init.go` (id=1, state, started_at, accumulated_ms, current_lap, total_laps, updated_at; seed stopped if absent)
- [x] 1.2 Implement race state model and elapsed logic (`accumulated_ms + (now - started_at)` while racing; pause folds elapsed into accumulated_ms; stop resets)
- [x] 1.3 Wire server state channel/mutex and lifecycle (started in `app/app.go` / `main.go`)

## 2. Backend: endpoints and broadcast

- [x] 2.1 Add `POST /api/race/state` (actions start/pause/resume/stop, auth controller + CSRF) and `GET /api/race/state` in `handlers/race_state.go` (or `handlers/race.go`)
- [x] 2.2 Register routes in `main.go` and add `RaceStateBroadcast` channel to server
- [x] 2.3 Implement `ws/ws.go` race_state broadcast channel/goroutine wrapping messages as `{type:'race_state', state, elapsed_ms, current_lap, total_laps}`; subscribe via `websocket-auth-rooms` topics when present else unconditional
- [x] 2.4 Implement 1s tick while racing only (start ticker on start/resume, stop on pause/stop; immediate broadcast on transition)

## 3. Backend: standings computation and broadcast

- [x] 3.1 Implement Go standings/gaps computation mirroring `computeGaps` (`ts/controller.ts:148`) semantics (LEAD / +N / empty)
- [x] 3.2 Broadcast `standings` on lap recorded and position change (event-driven, not on 1s tick) via `ws/ws.go`
- [x] 3.3 Hook standings recompute into lap-record write path

## 4. Frontend: synchronized views and race radio

- [x] 4.1 Update `ts/controller.ts` to consume `race_state`/`standings` broadcasts, render shared clock/lap/standings, and remove/reduce local `raceState`/`raceSeconds`/`currentLap` timer
- [x] 4.2 Update `ts/tv.ts`, `ts/spectator.ts`, `ts/pitboard.ts` to subscribe to `race_state`/`standings` and render shared clock, lap counter, and standings (no divergent client timers)
- [x] 4.3 Add TS race-radio receiver for `type:'race_radio'` in `ts/controller.ts`, `ts/driver.ts`/`ts/player.ts`, and `ts/pitboard.ts` and surface messages in UI (server already broadcasts)
- [x] 4.4 Reduce controller polling of `/api/lap-records` to fallback only (WS is primary for standings/state)

## 5. Tests and validation

- [x] 5.1 Go tests: state transitions (stopped→racing→paused→racing→stopped), invalid transitions, and controller-only auth rejection for spectator/player
- [x] 5.2 Go tests: persistence across simulated restart/reload (elapsed/lap preserved)
- [x] 5.3 Go tests: standings/gaps computation matches `computeGaps` fixtures and broadcasts on lap/position change
- [x] 5.4 Go/Playwright tests: `race_state` tick and transition broadcasts; race radio delivery to controller/driver/pitboard
- [x] 5.5 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
