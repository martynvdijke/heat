# trmnl-summary-api Specification

## Purpose
Public, unauthenticated `GET /api/trmnl/summary` endpoint returning a compact JSON document — the latest finalized race (finishing order with points) and the current season championship standings — optimized for TRMNL e-ink device polling.

## ADDED Requirements

### Requirement: Public summary endpoint
The system SHALL expose `GET /api/trmnl/summary` as a public, unauthenticated, read-only endpoint that returns JSON containing the latest race and the current season standings.

#### Scenario: Returns latest race and standings
- **WHEN** a client requests `GET /api/trmnl/summary`
- **THEN** the response SHALL have HTTP status 200 with `Content-Type: application/json`
- **AND** the response body SHALL contain a `latest_race` object and a `standings` array
- **AND** the response SHALL NOT require any authentication or session cookie

#### Scenario: Latest race shape
- **WHEN** at least one finalized round snapshot exists
- **THEN** `latest_race` SHALL include `name`, `race_date`, `round`, `country`, `track`, and `total_laps` (as available)
- **AND** `latest_race.results` SHALL be an array ordered by finishing `position` ascending, each entry containing `racer_name`, `position`, and `points`

#### Scenario: Standings shape
- **WHEN** a season with finalized round snapshots exists
- **THEN** `standings` SHALL be an array ordered by total `points` descending
- **AND** each entry SHALL contain `racer_name`, `races`, `wins`, and `points`

#### Scenario: No data present
- **WHEN** no finalized round snapshots or no seasons exist
- **THEN** the response SHALL still have HTTP status 200 with valid JSON
- **AND** `latest_race` SHALL be `null` and `standings` SHALL be an empty array

### Requirement: Latest race selection
The endpoint SHALL select the most recent finalized round snapshot as the latest race.

#### Scenario: Multiple finalized rounds
- **WHEN** several round snapshots have `status = 'final'`
- **THEN** the snapshot with the most recent `race_date` (tie-broken by highest `round`) SHALL be used as `latest_race`

#### Scenario: Only draft rounds
- **WHEN** round snapshots exist but none have `status = 'final'`
- **THEN** `latest_race` SHALL be `null`

#### Scenario: Results limit
- **WHEN** a finalized round has more than 10 racer results
- **THEN** `latest_race.results` SHALL contain at most the first 10 by `position`

### Requirement: Season selection for standings
The endpoint SHALL compute standings from the active season, falling back to the most recently created season when none is active.

#### Scenario: Active season exists
- **WHEN** a season with `status = 'active'` exists
- **THEN** `standings` SHALL be aggregated from that season's finalized round snapshots

#### Scenario: No active season
- **WHEN** no season has `status = 'active'`
- **THEN** the most recently created season SHALL be used for standings

#### Scenario: Standings limit
- **WHEN** more than 8 racers have standings points
- **THEN** the `standings` array SHALL contain at most the top 8 by `points`

### Requirement: Standings source consistency
Standings SHALL be aggregated exclusively from finalized round snapshots (`round_snapshots.status = 'final'`) joined to their `round_snapshot_scores`, mirroring the existing `RacerStatsBySeason` computation.

#### Scenario: Draft rounds excluded from standings
- **WHEN** a season contains both draft and finalized round snapshots
- **THEN** only scores from finalized snapshots SHALL contribute to `standings`

#### Scenario: Racer names included
- **WHEN** standings are returned
- **THEN** each entry SHALL include the racer name as stored on the round snapshot scores
