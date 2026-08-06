## ADDED Requirements

### Requirement: Round must have a season
Every round_snapshot SHALL have a non-null, valid season_id that references an existing season.

#### Scenario: Creating a round with valid season_id
- **WHEN** a POST request is made to `/api/rounds` with a valid `season_id`
- **THEN** the round is created successfully with that season_id

#### Scenario: Creating a round without season_id
- **WHEN** a POST request is made to `/api/rounds` without a `season_id`
- **THEN** the request is rejected with HTTP 400

#### Scenario: Creating a round with non-existent season_id
- **WHEN** a POST request is made to `/api/rounds` with a `season_id` that does not match any season
- **THEN** the request is rejected with HTTP 404

#### Scenario: Creating a round in an archived season
- **WHEN** a POST request is made to `/api/rounds` with a `season_id` of an archived season
- **THEN** the request is rejected with HTTP 409

### Requirement: Unique round numbers per season
Within a single season, round numbers SHALL be unique. The database SHALL enforce a UNIQUE constraint on `(season_id, round)`.

#### Scenario: Creating a round with duplicate round number
- **WHEN** a POST request is made to `/api/rounds` with a `season_id` and `round` that already exists in that season
- **THEN** the request is rejected with an appropriate error

#### Scenario: Auto-assigned round numbers are unique per season
- **WHEN** multiple rounds are created in the same season without specifying a round number
- **THEN** the system auto-assigns sequential round numbers unique within that season

### Requirement: Finalized rounds are immutable
Once a round snapshot has status `final`, its scores SHALL NOT be editable. Season and round number are also immutable.

#### Scenario: Editing a finalized round
- **WHEN** a PATCH request is made to `/api/rounds` for a finalized round
- **THEN** the request is rejected with HTTP 409

#### Scenario: Deleting a finalized round in an archived season
- **WHEN** a DELETE request is made to `/api/rounds` for a round in an archived season
- **THEN** the request is rejected with HTTP 409

### Requirement: Database-level constraints
The database SHALL enforce:
- `round_snapshots.season_id` is NOT NULL with a foreign key to `seasons.id` (ON DELETE CASCADE)
- A UNIQUE constraint on `(season_id, round)`

#### Scenario: Foreign key prevents orphan rounds
- **WHEN** a season is deleted
- **THEN** all its associated round snapshots and scores are cascade-deleted

#### Scenario: Unique constraint enforced at DB level
- **WHEN** an INSERT attempts to create a round with an existing `(season_id, round)` pair
- **THEN** the database rejects the insert
