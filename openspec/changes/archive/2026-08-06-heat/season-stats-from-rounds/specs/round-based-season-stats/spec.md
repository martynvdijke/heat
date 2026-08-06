## ADDED Requirements

### Requirement: Season stats aggregated from round snapshots

When a season_id parameter is provided to `/api/racer-stats`, the system SHALL aggregate driver statistics from `round_snapshot_scores` joined with `round_snapshots` where `status = 'final'`, rather than from `race_results`. The aggregated fields SHALL include: races, wins, gold, silver, bronze, points, DNF, DNS, spins, and overheated. Fastest laps SHALL be excluded from the round-based aggregation (returns 0).

#### Scenario: Season with finalized rounds returns aggregated snapshot stats
- **WHEN** a GET request is made to `/api/racer-stats?season_id=1` and season 1 has 3 finalized rounds with scores
- **THEN** the response SHALL contain one entry per racer with total points, races, wins, podiums, dnf, dns, spins, and overheated summed across all 3 rounds

#### Scenario: Season with no finalized rounds falls back to racer_stats
- **WHEN** a GET request is made to `/api/racer-stats?season_id=2` and season 2 has 0 finalized rounds
- **THEN** the response SHALL fall back to querying the `racer_stats` table

#### Scenario: Season with mixed finalized and draft rounds only counts finalized
- **WHEN** a GET request is made to `/api/racer-stats?season_id=3` and season 3 has 2 finalized rounds and 1 draft round
- **THEN** the response SHALL only include data from the 2 finalized rounds

### Requirement: Spins and overheated included in racer_stats queries

The `AllRacerStats()` and `SingleRacerStatsFallback()` functions SHALL include `spins` and `overheated` columns in their SELECT queries against the `racer_stats` table.

#### Scenario: All racer stats includes spins and overheated
- **WHEN** a GET request is made to `/api/racer-stats` (no season_id)
- **THEN** each racer stats object in the response SHALL include `spins` and `overheated` fields

#### Scenario: Single racer fallback includes spins and overheated
- **WHEN** a GET request is made to `/api/racer-stats?id=5` and the racer has a row in `racer_stats`
- **THEN** the response SHALL include `spins` and `overheated` fields in the stats object

### Requirement: Spins and overheated displayed on stats page

The Driver Performance table on the stats page SHALL display "Spins" and "Overheated" columns with the corresponding data from the API response.

#### Scenario: Driver stats table shows spins and overheated
- **WHEN** the stats page loads and renders the Driver Performance table
- **THEN** the table SHALL have "Spins" and "Overheated" column headers and each row SHALL show the driver's spins and overheated values

### Requirement: Driver share endpoint returns spins and overheated

The `/api/shared/driver-stats` endpoint SHALL include `spins` and `overheated` from the `racer_stats` table in its response.

#### Scenario: Driver share stats include spins and overheated
- **WHEN** a GET request is made to `/api/shared/driver-stats?token=valid_token`
- **THEN** the stats object in the response SHALL include nonzero `spins` and `overheated` fields if the racer has those values
