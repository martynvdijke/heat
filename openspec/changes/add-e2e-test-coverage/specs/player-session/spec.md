## ADDED Requirements

### Requirement: Player can log in
The system SHALL allow a player to log in via POST /api/player/login with racer_id and device_name. The system SHALL return a token, racer_id, and racer_name.

#### Scenario: Player login
- **WHEN** a POST /api/player/login request is sent with a valid racer_id and device_name
- **THEN** the response status SHALL be 200 OK
- **AND** the response SHALL contain "token", "racer_id", and "racer_name" fields

### Requirement: Player login fails for nonexistent racer
The system SHALL return 404 when a player attempts to log in with an invalid racer_id.

#### Scenario: Login with invalid racer ID
- **WHEN** a POST /api/player/login request is sent with an invalid racer_id (e.g., 9999)
- **THEN** the response status SHALL be 404

### Requirement: Player token can be validated
The system SHALL validate a player token via GET /api/player/validate with X-Player-Token header and return the associated racer_id and racer_name.

#### Scenario: Validate valid token
- **WHEN** a player logs in to obtain a token
- **AND** a GET /api/player/validate request is sent with that token in the X-Player-Token header
- **THEN** the response status SHALL be 200 OK
- **AND** the response SHALL contain "racer_id" and "racer_name"

### Requirement: Invalid player token is rejected
The system SHALL return 401 when an invalid or expired token is used to validate a player session.

#### Scenario: Reject invalid token
- **WHEN** a GET /api/player/validate request is sent with X-Player-Token set to an invalid value
- **THEN** the response status SHALL be 401

### Requirement: Player can report gear
The system SHALL accept gear shift reports from a player via POST /api/player/gear with token, lap, gear, and stress fields.

#### Scenario: Report gear shift
- **WHEN** a logged-in player sends POST /api/player/gear with lap, gear, and stress
- **THEN** the response status SHALL be 200 OK

### Requirement: Player can report heat card usage
The system SHALL accept heat card usage reports from a player via POST /api/player/heat with token, card_type, location, and count fields.

#### Scenario: Report heat usage
- **WHEN** a logged-in player sends POST /api/player/heat with card_type, location, and count
- **THEN** the response status SHALL be 200 OK

### Requirement: Player can report turbo usage
The system SHALL accept turbo usage reports from a player via POST /api/player/turbo with token and lap fields.

#### Scenario: Report turbo usage
- **WHEN** a logged-in player sends POST /api/player/turbo with lap
- **THEN** the response status SHALL be 200 OK

### Requirement: Player can get their status
The system SHALL return a player's current status (racer info and heat cards) via GET /api/player/status with X-Player-Token header.

#### Scenario: Get player status
- **WHEN** a logged-in player sends GET /api/player/status with their token
- **THEN** the response status SHALL be 200 OK
- **AND** the response SHALL contain "racer" and "heat_cards" fields
- **AND** the "racer" object SHALL contain a "name" field

### Requirement: Player can log out
The system SHALL invalidate a player's session (delete from player_sessions) via POST /api/player/logout with X-Player-Token header.

#### Scenario: Player logout
- **WHEN** a logged-in player sends POST /api/player/logout with their token
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/player/validate with the same token SHALL return 401
