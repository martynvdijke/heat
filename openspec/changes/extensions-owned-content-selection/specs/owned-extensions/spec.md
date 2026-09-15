## ADDED Requirements

### Requirement: Owned extensions are stored as a persistent membership set
The system SHALL store the set of extensions the group owns in an `owned_extensions` table, SHALL seed the Base Game extension as owned, and SHALL always treat the Base Game extension as owned regardless of the stored set.

#### Scenario: Base Game is owned by default
- **WHEN** the database is initialized
- **THEN** the Base Game extension (is_base=1) is present in the owned set

#### Scenario: Base Game can never be removed from the owned set
- **WHEN** a client attempts to replace the owned set without the Base Game id
- **THEN** the stored set still contains the Base Game id

### Requirement: Admin can manage which extensions are owned
The system SHALL provide `GET /api/extensions/owned` returning the owned extension ids, and `PUT /api/extensions/owned` replacing the owned set in full (Base Game force-added, unknown ids ignored).

#### Scenario: Retrieve the owned set
- **WHEN** a client requests `GET /api/extensions/owned`
- **THEN** the response includes the Base Game id and any other owned extension ids

#### Scenario: Mark an extension as owned
- **WHEN** a client sends `PUT /api/extensions/owned` including an extension id that was not owned
- **THEN** that extension is returned by `GET /api/extensions/owned`

#### Scenario: Remove an extension from the owned set
- **WHEN** a client sends `PUT /api/extensions/owned` without an extension id that was previously owned
- **THEN** that extension is no longer returned by `GET /api/extensions/owned`

#### Scenario: Unknown ids in a full replace are ignored
- **WHEN** a client sends `PUT /api/extensions/owned` including a non-existent extension id
- **THEN** the request succeeds and only existing extension ids are stored

### Requirement: Content ownership is derived from the extension set
Content SHALL be considered owned when its extension_id is in the owned set or when its extension_id is 0 (Base Game content). Content assigned to an extension that is not owned SHALL be considered not owned.

#### Scenario: Unassigned content counts as owned
- **WHEN** content has extension_id = 0
- **THEN** it is treated as owned content

#### Scenario: Content from an owned extension is owned
- **WHEN** content is assigned to an extension present in the owned set
- **THEN** it is treated as owned content

#### Scenario: Content from an unowned extension is not owned
- **WHEN** content is assigned to an extension not present in the owned set
- **THEN** it is not treated as owned content
