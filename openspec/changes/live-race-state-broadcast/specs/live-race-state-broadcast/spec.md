## ADDED Requirements

### Requirement: Authoritative race state
The system SHALL maintain a server-authoritative race state machine with states `stopped | racing | paused`, elapsed time, current lap, and total laps as the single source of truth, replacing the client-local `raceState`/`raceSeconds`/`currentLap` in `ts/controller.ts`.

#### Scenario: Start moves stopped to racing and begins elapsed
- **WHEN** an authenticated controller starts the race from `stopped`
- **THEN** the server transitions to `racing`, records the start time, and broadcasts `race_state` with `state: racing` and increasing `elapsed_ms`

#### Scenario: Pause freezes elapsed
- **WHEN** the controller pauses while `racing`
- **THEN** the server transitions to `paused`, freezes `elapsed_ms` (folding elapsed into `accumulated_ms`), and broadcasts `race_state` with `state: paused` and constant `elapsed_ms` until resumed

#### Scenario: Stop resets race
- **WHEN** the controller stops the race from `racing` or `paused`
- **THEN** the server transitions to `stopped`, resets `elapsed_ms` to 0 and `current_lap` to 0, and broadcasts `race_state` with `state: stopped`

### Requirement: Controller-only transitions
Only an authenticated controller SHALL be permitted to change race state; requests from other roles SHALL be rejected.

#### Scenario: Spectator or player cannot start the race
- **WHEN** a spectator or player issues `POST /api/race/state` with action `start`
- **THEN** the server responds with 401 or 403 and the race state remains unchanged

### Requirement: Race state persistence
The system SHALL persist race state so it survives server restart and page reload.

#### Scenario: State survives restart while racing
- **WHEN** the server restarts while the race is `racing` with a given `elapsed_ms` and `current_lap`
- **THEN** `GET /api/race/state` returns the same `current_lap` and `total_laps` and an `elapsed_ms` that reflects the persisted elapsed (continuing or resumed per elapsed computation), and clients can continue from the persisted state

### Requirement: race_state broadcast
The system SHALL broadcast a `race_state` message containing `state`, `elapsed_ms`, `current_lap`, and `total_laps` on every transition and on a 1s tick while `racing`.

#### Scenario: Transition broadcasts immediately
- **WHEN** the race state transitions (start, pause, resume, or stop)
- **THEN** the server immediately broadcasts a `race_state` message with the new state and current `elapsed_ms`, `current_lap`, and `total_laps`

#### Scenario: Racing ticks once per second
- **WHEN** the race is `racing`
- **THEN** the server broadcasts `race_state` approximately once per second with increasing `elapsed_ms`

#### Scenario: No ticks while stopped or paused
- **WHEN** the race is `stopped` or `paused`
- **THEN** the server does not emit periodic `race_state` ticks (only transition broadcasts)

### Requirement: Server-computed standings and gaps
The system SHALL compute positions and lap gaps server-side from racers and lap records using the existing `computeGaps` semantics (`ts/controller.ts:148`) and SHALL broadcast `standings` on lap or position changes.

#### Scenario: Recording a lap broadcasts updated gaps
- **WHEN** a lap is recorded that changes gaps
- **THEN** the server broadcasts `standings` where the leader gap is `LEAD`, lapped racers show `+N`, and same-lap racers show empty per `computeGaps` semantics

#### Scenario: Position change reorders standings
- **WHEN** a lap record causes a position change
- **THEN** the server broadcasts `standings` with racers reordered by position

### Requirement: Synchronized views
TV, spectator, pitboard, and controller views SHALL render the shared clock, lap counter, and standings from `race_state` and `standings` broadcasts without divergent client timers.

#### Scenario: TV and controller show same clock and lap
- **WHEN** `race_state` and `standings` broadcasts are received
- **THEN** the TV and controller pages display the same `elapsed_ms` clock value and the same `current_lap`/`total_laps`

### Requirement: Race radio delivery
The system SHALL consume the server's existing `race_radio` broadcast (`ws/ws.go:158-170`) on the controller, driver, and pitboard pages and surface messages in the UI.

#### Scenario: Race radio message appears on target pages
- **WHEN** the server broadcasts a `race_radio` message
- **THEN** the controller, driver, and pitboard pages display that message without a page reload
