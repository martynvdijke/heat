# internal-code-quality Specification

## Purpose
TBD - created by archiving change fix-race-report-and-split-monoliths. Update Purpose after archive.
## Requirements
### Requirement: Race report frontend matches backend data contract

The race report page and API SHALL use consistent field names, and the API SHALL return all fields the frontend renders.

#### Scenario: Race report API returns correct fields

- **WHEN** GET /api/race-report is called
- **THEN** the response SHALL include `name`, `track`, `country`, `total_laps`, `race_date`, and `results` fields
- **THEN** each result SHALL include `position`, `racer_name`, `points`, `fastest_lap`, `finished`, `dnf`, and `dns` fields

#### Scenario: Race report page displays all data correctly

- **WHEN** a user navigates to /race-report.html
- **THEN** the page SHALL display the race name, circuit, country, laps, date, and full results table
- **THEN** the page SHALL NOT auto-trigger the browser print dialog

### Requirement: Streaks function uses single query

The Streaks() function SHALL compute wins, podiums, total races, and average position in a single SQL query.

#### Scenario: Streaks() returns correct data

- **WHEN** Streaks() is called for a racer
- **THEN** the function SHALL execute exactly one SQL query that returns wins, podiums, total races, and average position

### Requirement: Monolith files are split by concern

The four identified monolith files SHALL be restructured into focused files, each containing a single domain or concern.

#### Scenario: racing/ contains multiple focused files

- **WHEN** listing files in the racing/ directory
- **THEN** racing/ SHALL contain separate files for stats, track stats, streaks, ELO, qualifying, consistency, head-to-head, CSV export, and race report

#### Scenario: handlers/ contains separate stats files

- **WHEN** listing files in the handlers/ directory
- **THEN** stats-related handler functions SHALL be organized into domain-grouped files rather than a single stats.go

#### Scenario: middleware/ contains separate files

- **WHEN** listing files in the middleware/ directory
- **THEN** each middleware concern (security, CSRF, request ID, rate limiting, auth, Umami) SHALL have its own file

#### Scenario: db/ contains separate files

- **WHEN** listing files in the db/ directory
- **THEN** db/ SHALL contain separate files for initialization, backup, seed data, and helpers

### Requirement: Remove duplicate functions

Duplicate or legacy functions SHALL be consolidated into single canonical implementations.

#### Scenario: RacerStatsFallback is removed

- **WHEN** searching for RacerStatsFallback in the codebase
- **THEN** it SHALL NOT exist (replaced by AllRacerStats)

#### Scenario: SingleRacerStatsFallback is removed

- **WHEN** searching for SingleRacerStatsFallback in the codebase
- **THEN** it SHALL NOT exist (replaced by SingleRacerStatsBySeason)

