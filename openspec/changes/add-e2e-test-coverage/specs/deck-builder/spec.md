## ADDED Requirements

### Requirement: Available upgrades can be queried for a racer
The system SHALL return upgrades that a racer has not yet purchased via GET /api/available-upgrades filtered by racer_id.

#### Scenario: Get available upgrades
- **WHEN** a GET /api/available-upgrades request is sent with a racer_id parameter
- **THEN** the response status SHALL be 200 OK
- **AND** the response SHALL be a JSON array of upgrade card objects

### Requirement: Admin can buy an upgrade for a racer
The system SHALL allow purchasing an upgrade for a racer via POST /api/player-upgrades/buy with racer_id, upgrade_id, season_id, and round.

#### Scenario: Buy upgrade
- **WHEN** an admin sends POST /api/player-upgrades/buy with racer_id, upgrade_id, season_id, round
- **THEN** the response status SHALL be 200 OK

### Requirement: Player upgrades can be queried
The system SHALL return a racer's purchased upgrades via GET /api/player-upgrades filtered by racer_id.

#### Scenario: Get player upgrades
- **WHEN** a GET /api/player-upgrades request is sent with a racer_id parameter
- **THEN** the response SHALL be a JSON array of player upgrade objects, each with upgrade details

### Requirement: Upgrade equipped status can be toggled
The system SHALL allow toggling a player upgrade's equipped status via PUT /api/player-upgrades/toggle with id and equipped fields.

#### Scenario: Toggle upgrade equipped
- **WHEN** an admin sends PUT /api/player-upgrades/toggle with an existing upgrade id and equipped: false
- **THEN** the response status SHALL be 200 OK

### Requirement: Legend abilities can be assigned to racers
The system SHALL allow assigning legend abilities to racers via POST /api/legend-abilities/assign with racer_id and ability_id.

#### Scenario: Assign legend ability
- **WHEN** an admin sends POST /api/legend-abilities/assign with a valid racer_id and ability_id
- **THEN** the response status SHALL be 200 OK

### Requirement: Racer legend abilities can be queried
The system SHALL return racer-specific legend abilities via GET /api/racer-legend-abilities filtered by racer_id.

#### Scenario: Get racer legend abilities
- **WHEN** a GET /api/racer-legend-abilities request is sent with a racer_id parameter
- **THEN** the response SHALL be a JSON array of racer legend ability objects
