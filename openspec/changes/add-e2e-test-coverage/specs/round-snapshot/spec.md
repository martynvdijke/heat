## ADDED Requirements

### Requirement: Admin can take a round snapshot
The system SHALL allow creating a round snapshot via POST /api/rounds that captures current racer standings (points, positions) for a season round.

#### Scenario: Take round snapshot with default round number
- **WHEN** an admin sends POST /api/rounds with {race_name: "Test Round", season_id: 1}
- **THEN** the response SHALL include an "id" field and a "round" field
- **AND** the round number SHALL be auto-incremented

#### Scenario: Take round snapshot with explicit round number
- **WHEN** an admin sends POST /api/rounds with {race_name: "Explicit Round", season_id: 1, round: 5}
- **THEN** the response SHALL include the round number 5

### Requirement: Round snapshots can be retrieved
The system SHALL return round snapshots via GET /api/rounds, optionally filtered by id or season_id.

#### Scenario: Get all snapshots
- **WHEN** a GET /api/rounds request is sent
- **THEN** the response SHALL be a JSON array of snapshots, each with id, season_id, race_name, race_date, round, created_at

#### Scenario: Get snapshot by ID
- **WHEN** a GET /api/rounds request is sent with an existing snapshot id
- **THEN** the response SHALL be a single snapshot object with scores array containing racer_id, racer_name, points, position

### Requirement: Round snapshots can be deleted
The system SHALL delete a round snapshot and its associated scores via DELETE /api/rounds with a valid id.

#### Scenario: Delete snapshot
- **WHEN** a DELETE /api/rounds request is sent with an existing snapshot id
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/rounds with that id SHALL return 404
