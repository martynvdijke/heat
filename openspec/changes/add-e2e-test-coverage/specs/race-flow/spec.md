## ADDED Requirements

### Requirement: Race info can be updated and retrieved
The system SHALL allow updating race info (country, track, track_id, laps) via POST /api/race-info and retrieving it via GET /api/race-info. Updated values SHALL persist.

#### Scenario: Update race info
- **WHEN** an admin sends POST /api/race-info with {country: "Belgium", track: "Spa", track_id: "spa", laps: 44}
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/race-info SHALL return the updated country, track, and laps

### Requirement: A full race can be saved to history
The system SHALL accept a completed race via POST /api/race-history with results including racer_id, racer_name, position, points, fastest_lap, finished, and did_not_start fields. A successful save SHALL return the race ID.

#### Scenario: Save completed race with multiple racers
- **WHEN** a POST /api/race-history is sent with race_name, country, track, track_id, total_laps, race_type, and an array of results with position, points, fastest_lap
- **THEN** the response SHALL include an "id" field for the created race history entry

### Requirement: Race history can be retrieved
The system SHALL return race history entries via GET /api/race-history ordered by date descending.

#### Scenario: Get race history list
- **WHEN** a GET /api/race-history request is sent
- **THEN** the response SHALL be a JSON array
- **AND** if races have been saved, the array SHALL contain at least one entry

### Requirement: Race can be deleted from history
The system SHALL delete a race history entry and its associated race results when DELETE /api/race-history is called with a valid id.

#### Scenario: Delete race history
- **WHEN** a DELETE /api/race-history request is sent with an existing race id
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/race-history with that id SHALL return an empty array

### Requirement: One-off races can be created and queried
The system SHALL support creating races with race_type "oneoff" and querying them via GET /api/oneoff-races.

#### Scenario: Create and list one-off races
- **WHEN** a POST /api/race-history is sent with race_type "oneoff"
- **THEN** the response SHALL include an "id" field
- **AND** a GET /api/oneoff-races SHALL return a JSON array including the new race
- **AND** deleting a one-off race via DELETE /api/oneoff-races SHALL return 200

### Requirement: One-off race does not update racer stats
The system SHALL NOT update racer_stats (races count, wins, gold, etc.) when a race with race_type "oneoff" is saved.

#### Scenario: One-off race stats isolation
- **WHEN** a race is saved with race_type "oneoff" and results
- **AND** the winning racer's stats are queried via GET /api/racer-stats
- **THEN** the racer's races count SHALL remain unchanged (not incremented)
