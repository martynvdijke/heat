## ADDED Requirements

### Requirement: Admin can set the next race date

The system SHALL allow an authenticated admin to set and clear the date of the next race event from the Race Day tab in the admin panel. The date SHALL be stored as a `YYYY-MM-DD` string and persist across server restarts. An empty value SHALL clear the stored date. Invalid date formats SHALL be rejected with an error and MUST NOT be stored.

#### Scenario: Admin sets a valid next race date

- **WHEN** an authenticated admin submits a valid date (e.g., `2026-08-22`) via the race info form on the Race Day tab
- **THEN** the system stores the date and the stored value matches the submitted date

#### Scenario: Admin clears the next race date

- **WHEN** an authenticated admin submits an empty date via the race info form on the Race Day tab
- **THEN** the system clears the stored date and the race info response contains an empty next race date

#### Scenario: Admin submits an invalid date

- **WHEN** an authenticated admin submits a value that is not a valid `YYYY-MM-DD` date (e.g., `not-a-date`)
- **THEN** the system rejects the request with a `400` response and MUST NOT change the stored date

### Requirement: Next race date is retrievable via the race info API

The system SHALL return the stored next race date in the `GET /api/race-info` response under the `next_race_date` field. When no date has been set, the field SHALL be an empty string.

#### Scenario: Retrieve a configured next race date

- **WHEN** a client calls `GET /api/race-info` and a next race date has been stored
- **THEN** the response contains the stored date in the `next_race_date` field

#### Scenario: Retrieve when no next race date is configured

- **WHEN** a client calls `GET /api/race-info` and no next race date has been stored
- **THEN** the response contains an empty string in the `next_race_date` field

### Requirement: Next race date survives restarts

The system SHALL persist the next race date in the database such that the value is retained when the server restarts.

#### Scenario: Date persists across server restart

- **WHEN** an admin has set a next race date and the server restarts
- **THEN** the `GET /api/race-info` response still contains the previously stored date in the `next_race_date` field
