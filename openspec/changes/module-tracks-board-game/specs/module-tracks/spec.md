## ADDED Requirements

### Requirement: Tracks attributed to modules
The system SHALL allow each track to optionally belong to a gameplay module via a `module_id` column on the `tracks` table (0 = not module-specific). `GET /api/tracks` and `POST /api/tracks` SHALL include and persist `module_id`. When a track is assigned to a module, its `extension_id` SHALL be derived from the module's owning extension so the extension catalog stays consistent.

#### Scenario: Saving a track with a module
- **WHEN** an admin saves a track with `module_id` set to an existing module
- **THEN** the track is persisted with that `module_id` and its `extension_id` matches the module's owning extension

#### Scenario: Saving a track without a module
- **WHEN** an admin saves a track with no `module_id` (0)
- **THEN** the track keeps its explicitly set `extension_id` and `module_id = 0`

#### Scenario: Track list returns module attribution
- **WHEN** an admin GETs `/api/tracks`
- **THEN** each track in the response includes its `module_id`

### Requirement: Track editor module selector
The track editor (admin modal and htmx edit form) SHALL offer a Module dropdown listing all modules grouped by owning extension. Saving the form SHALL persist the chosen module and derive the extension as defined above.

#### Scenario: Admin assigns a track to a module
- **WHEN** an admin opens the track editor and selects a module from the Module dropdown
- **THEN** the track's `module_id` is persisted and its extension is updated to the module's extension

### Requirement: Extension catalog shows module tracks
The extension detail view SHALL list each module with the tracks that belong to it (via `module_id`), alongside the existing extension-level track assignment table.

#### Scenario: Admin views an extension with module tracks
- **WHEN** an admin opens an extension's detail view and that extension has modules which own tracks
- **THEN** each module row lists its tracks, and the extension-level Tracks table still lists all of the extension's tracks

#### Scenario: Module deletion resets its tracks
- **WHEN** an admin deletes a module that owns tracks
- **THEN** those tracks have `module_id = 0` and their `extension_id` is reset to the Base Game extension
