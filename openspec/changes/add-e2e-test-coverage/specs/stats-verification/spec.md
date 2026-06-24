## ADDED Requirements

### Requirement: Race results update gold/silver/bronze counts
The system SHALL increment racer_stats.gold for position 1, racer_stats.silver for position 2, and racer_stats.bronze for position 3 when a season race is saved.

#### Scenario: Gold/silver/bronze after race
- **WHEN** a season race is saved with racers finishing in positions 1, 2, 3, 4, 5
- **AND** the stats for each racer are queried via GET /api/racer-stats
- **THEN** position 1 SHALL have gold=1, silver=0, bronze=0, wins=1
- **AND** position 2 SHALL have gold=0, silver=1, bronze=0, wins=0
- **AND** position 3 SHALL have gold=0, silver=0, bronze=1, wins=0
- **AND** position 4+ SHALL have gold=0, silver=0, bronze=0, wins=0

### Requirement: DNF and DNS are tracked in racer stats
The system SHALL increment racer_stats.dnf when a racer's finished=false and did_not_start=false. It SHALL increment racer_stats.dns when did_not_start=true.

#### Scenario: DNF and DNS tracking
- **WHEN** a season race is saved with one racer having finished=false, did_not_start=false (DNF)
- **AND** another racer having did_not_start=true (DNS)
- **AND** stats are queried for each racer
- **THEN** the DNF racer SHALL have dnf=1, dns=0
- **AND** the DNS racer SHALL have dnf=0, dns=1

### Requirement: Racer stats can be retrieved
The system SHALL return racer statistics via GET /api/racer-stats including wins, gold, silver, bronze, races, fastest_laps, dnf, dns.

#### Scenario: Get racer stats
- **WHEN** a GET /api/racer-stats request is sent with a valid racer id
- **THEN** the response SHALL contain a "stats" object with the racer's aggregate statistics

### Requirement: Head-to-head comparison works
The system SHALL return head-to-head statistics between two racers via GET /api/stats/head-to-head.

#### Scenario: Head-to-head query
- **WHEN** a GET /api/stats/head-to-head request is sent with racer1 and racer2 parameters
- **THEN** the response SHALL include racer1_name and racer2_name fields

### Requirement: Points progression can be retrieved
The system SHALL return a racer's points progression across races via GET /api/stats/points-progression.

#### Scenario: Points progression query
- **WHEN** a GET /api/stats/points-progression request is sent with a racer_id parameter
- **THEN** the response SHALL be a JSON array of progression data points

### Requirement: Streaks data is available
The system SHALL return streak information (win streaks, podium streaks, etc.) via GET /api/stats/streaks.

#### Scenario: Get streaks
- **WHEN** a GET /api/stats/streaks request is sent
- **THEN** the response SHALL be a JSON array of streak info objects

### Requirement: ELO ratings are available
The system SHALL return ELO ratings for all racers via GET /api/stats/elo.

#### Scenario: Get ELO ratings
- **WHEN** a GET /api/stats/elo request is sent
- **THEN** the response SHALL be a JSON array of ELO rating objects

### Requirement: Stats can be exported as CSV
The system SHALL export racer statistics as CSV via GET /api/stats/export with Content-Type text/csv.

#### Scenario: Export stats CSV
- **WHEN** a GET /api/stats/export request is sent
- **THEN** the response Content-Type SHALL be "text/csv"
- **AND** the response body SHALL contain CSV headers including "Name"
- **AND** the response body SHALL contain "Gold", "Silver", "Bronze" columns

### Requirement: Track performance stats are available
The system SHALL return track performance data via GET /api/stats/track-performance, optionally filtered by racer_id.

#### Scenario: Track performance for all racers
- **WHEN** a GET /api/stats/track-performance request is sent
- **THEN** the response status SHALL be 200 OK

#### Scenario: Track performance for specific racer
- **WHEN** a GET /api/stats/track-performance request is sent with a racer_id parameter
- **THEN** the response status SHALL be 200 OK
