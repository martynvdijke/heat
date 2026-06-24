## ADDED Requirements

### Requirement: Heat cards can be initialized for racers
The system SHALL initialize heat card decks for given racers via POST /api/heat-cards/init-decks with race_id and racer_ids. Each racer SHALL receive 7 heat cards in their deck with 3 dealt to hand.

#### Scenario: Initialize heat decks
- **WHEN** an admin sends POST /api/heat-cards/init-decks with race_id and an array of racer_ids
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/heat-cards SHALL return at least 21 heat cards (7 × 3 racers)

### Requirement: Heat cards can be queried
The system SHALL return heat cards via GET /api/heat-cards, optionally filtered by racer_id.

#### Scenario: Get heat cards for racer
- **WHEN** a GET /api/heat-cards request is sent with a racer_id parameter
- **THEN** the response SHALL be a JSON array of heat card objects

### Requirement: Heat cards can be added
The system SHALL allow adding a heat card via POST /api/heat-cards with racer_id, location, card_type, lap_added.

#### Scenario: Add heat card
- **WHEN** a POST /api/heat-cards request is sent with valid heat card data
- **THEN** the response status SHALL be 200 OK

### Requirement: Heat cards can be moved between locations
The system SHALL allow moving a heat card between locations (deck, hand, engine, discard) via PUT /api/heat-cards/move.

#### Scenario: Move heat card
- **WHEN** a PUT /api/heat-cards/move request is sent with card_id and a new location
- **THEN** the response status SHALL be 200 OK

### Requirement: Heat cards can be deleted
The system SHALL delete a heat card via DELETE /api/heat-cards with a valid id.

#### Scenario: Delete heat card
- **WHEN** a DELETE /api/heat-cards request is sent with a valid card id
- **THEN** the response status SHALL be 200 OK

### Requirement: All heat cards can be cleared
The system SHALL clear all heat cards (or for a specific racer) via DELETE /api/heat-cards/clear.

#### Scenario: Clear all heat cards
- **WHEN** a DELETE /api/heat-cards/clear request is sent
- **THEN** the response status SHALL be 200 OK

### Requirement: Gear shifts can be recorded and queried
The system SHALL accept gear shift data via POST /api/gear-shifts with racer_id, race_id, lap, gear, stress. The system SHALL return gear shifts via GET /api/gear-shifts filtered by racer_id.

#### Scenario: Get gear shifts
- **WHEN** a GET /api/gear-shifts request is sent with a racer_id parameter
- **THEN** the response SHALL be a JSON array of gear shift objects

### Requirement: Upgrade cards are available
The system SHALL return available upgrade cards via GET /api/upgrade-cards.

#### Scenario: Get upgrade cards
- **WHEN** a GET /api/upgrade-cards request is sent
- **THEN** the response SHALL be a JSON array of upgrade card objects
- **AND** the array SHALL contain at least 8 upgrade cards (matching the seed data)

### Requirement: Legend abilities are available
The system SHALL return legend abilities via GET /api/legend-abilities.

#### Scenario: Get legend abilities
- **WHEN** a GET /api/legend-abilities request is sent
- **THEN** the response SHALL be a JSON array of legend ability objects
- **AND** the array SHALL contain at least 5 legend abilities (matching the seed data)

### Requirement: Lap records can be created and queried
The system SHALL accept lap records via POST /api/lap-records with race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used. The system SHALL return lap records via GET /api/lap-records filtered by race_id.

#### Scenario: Record and get lap records
- **WHEN** a POST /api/lap-records request is sent with valid lap record data
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/lap-records with the same race_id SHALL return at least one record

### Requirement: Batch lap records can be submitted
The system SHALL accept batch lap records via POST /api/lap-records/batch with race_id, lap, and an array of records each containing racer_id, position, gear_used, heat_generated, turbo_used.

#### Scenario: Submit batch lap records
- **WHEN** a POST /api/lap-records/batch request is sent with a lap number and multiple racer records
- **THEN** the response status SHALL be 200 OK

### Requirement: Sectors data is available
The system SHALL return sector information for a track via GET /api/sectors.

#### Scenario: Get sectors for Monza
- **WHEN** a GET /api/sectors request is sent with track_id "monza"
- **THEN** the response SHALL be a JSON array of sector objects
- **AND** the array SHALL contain at least 5 sectors

### Requirement: Race events can be recorded and queried
The system SHALL accept race events (overtakes, crashes, etc.) via POST /api/race-events with race_id, lap, event_type, racer_id, racer_id2, note. The system SHALL return events via GET /api/race-events.

#### Scenario: Add and list race events
- **WHEN** a POST /api/race-events request is sent with event_type "overtake", racer_id, racer_id2, note
- **THEN** the response status SHALL be 200 OK
- **AND** a subsequent GET /api/race-events with the same race_id SHALL return at least one event
- **AND** the first event's event_type SHALL be "overtake"

### Requirement: AI difficulty defaults are available
The system SHALL return AI difficulty defaults via GET /api/ai-difficulty including difficulty, aggression, error_rate, and consistency.

#### Scenario: Get AI difficulty defaults
- **WHEN** a GET /api/ai-difficulty request is sent
- **THEN** the response SHALL contain "difficulty", "aggression", "error_rate", and "consistency" fields

### Requirement: AI difficulty can be set
The system SHALL accept AI difficulty settings via POST /api/ai-difficulty.

#### Scenario: Set AI difficulty
- **WHEN** an admin sends POST /api/ai-difficulty with difficulty, aggression, error_rate, consistency
- **THEN** the response status SHALL be 200 OK
