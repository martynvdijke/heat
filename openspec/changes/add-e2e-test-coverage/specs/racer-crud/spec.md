## ADDED Requirements

### Requirement: Admin can create a racer via API
The system SHALL allow an authenticated admin to create a new racer via POST /api/racers with name, car_name, car_color, profile_picture, points, rank, and position fields. The created racer SHALL appear in the GET /api/racers response and on the leaderboard.

#### Scenario: Create racer with all fields
- **WHEN** an admin sends POST /api/racers with valid JSON body including name, car_name, car_color, points, rank, position
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/racers SHALL include the new racer in the list

### Requirement: Admin can edit a racer via API
The system SHALL allow updating an existing racer's fields via POST /api/racers with the racer's id included. Updated fields SHALL be reflected in subsequent GET /api/racers responses.

#### Scenario: Edit racer name
- **WHEN** an admin sends POST /api/racers with an existing racer id and a new name
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/racers SHALL show the updated name for that racer

### Requirement: Admin cannot create a racer without authentication
The system SHALL reject unauthenticated POST /api/racers requests with 401 Unauthorized.

#### Scenario: Unauthenticated racer creation fails
- **WHEN** a request is sent to POST /api/racers without a valid session cookie
- **THEN** the response status SHALL be 401

### Requirement: Created racer appears on leaderboard
The system SHALL display newly created racers on the index page leaderboard.

#### Scenario: New racer visible on leaderboard
- **WHEN** an admin creates a new racer via the API
- **AND** the index page is loaded
- **THEN** the leaderboard SHALL contain a row with the new racer's name

### Requirement: Admin can delete a racer
The system SHALL allow deleting a racer via the admin UI's racer delete button, which triggers an HTMX DELETE request. The racer SHALL be removed from the racers list after deletion.

#### Scenario: Delete racer from admin UI
- **WHEN** an admin clicks the delete button for a racer in the racers tab
- **AND** confirms the deletion dialog
- **THEN** the racer SHALL no longer appear in the racer list table
