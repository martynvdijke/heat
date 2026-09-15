## ADDED Requirements

### Requirement: Race-day track selection only offers owned tracks
The system SHALL restrict the tracks offered in the race-day track select (admin tab and controller) to owned tracks. `GET /api/tracks?owned=1` SHALL return only tracks whose extension is owned (or extension_id = 0); without the parameter the full list SHALL be returned unchanged.

#### Scenario: Admin track select excludes unowned tracks
- **WHEN** an admin opens the race-day tab and `GET /api/tracks?owned=1` is used to populate the track select
- **THEN** tracks assigned to unowned extensions are absent from the select
- **AND** owned tracks and Base Game tracks are present

#### Scenario: Controller track select excludes unowned tracks
- **WHEN** the controller page loads its track select from owned tracks
- **THEN** tracks assigned to unowned extensions are absent
- **AND** owned tracks and Base Game tracks are present

#### Scenario: Full track list remains available without the owned filter
- **WHEN** a client requests `GET /api/tracks` without the owned parameter
- **THEN** all tracks are returned, including those from unowned extensions

### Requirement: Upgrade deck-builder only offers owned upgrades
The system SHALL restrict `GET /api/available-upgrades` to upgrades from owned extensions (or extension_id = 0).

#### Scenario: Deck-builder excludes unowned upgrades
- **WHEN** a client requests `GET /api/available-upgrades` for a racer
- **THEN** the response contains no upgrades assigned to unowned extensions
- **AND** owned and Base Game upgrades are still offered

### Requirement: Legend assignment catalog only offers owned legends
The system SHALL restrict `GET /api/legend-abilities` to legends from owned extensions (or extension_id = 0).

#### Scenario: Legend catalog excludes unowned legends
- **WHEN** a client requests `GET /api/legend-abilities`
- **THEN** the response contains no legends assigned to unowned extensions
- **AND** owned and Base Game legends are still offered

### Requirement: Assigned legends and purchased upgrades always remain visible
The system SHALL NOT filter already-assigned or already-purchased content by ownership. Legends assigned to a racer (via `GET /api/racer-legend-abilities`) and upgrades the racer owns SHALL remain fully visible regardless of the owning extension's owned status, so assigned content can always be reviewed and unassigned.

#### Scenario: Assigned legend from an unowned extension stays visible
- **WHEN** a racer has a legend assigned that belongs to an extension later marked unowned
- **THEN** the legend still appears in `GET /api/racer-legend-abilities` for that racer
- **AND** it can still be unassigned

#### Scenario: Purchased upgrade from an unowned extension stays visible
- **WHEN** a racer has purchased an upgrade from an extension later marked unowned
- **THEN** the upgrade still appears in the racer's purchased upgrades
