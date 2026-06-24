## ADDED Requirements

### Requirement: Weather conditions can be set
The system SHALL accept weather condition data via POST /api/weather with race_id, condition, lap_start, lap_end, and grip_modifier fields. The handler SHALL insert a new record.

#### Scenario: Set weather condition
- **WHEN** an admin sends POST /api/weather with race_id, condition "wet", lap_start 1, lap_end 999, grip_modifier 0.7
- **THEN** the response status SHALL be 200 OK

### Requirement: Weather conditions can be retrieved
The system SHALL return weather conditions for a race via GET /api/weather filtered by race_id. The response SHALL include condition, lap_start, lap_end, grip_modifier.

#### Scenario: Get weather for race
- **WHEN** a GET /api/weather request is sent with race_id
- **THEN** the response status SHALL be 200 OK
- **AND** the response SHALL be a JSON array of weather conditions
- **AND** if weather was previously set, the array SHALL contain at least one entry

### Requirement: Weather conditions can be deleted
The system SHALL delete a weather condition record via DELETE /api/weather with a valid id.

#### Scenario: Delete weather condition
- **WHEN** a DELETE /api/weather request is sent with a valid weather id
- **THEN** the response status SHALL be 200 OK
