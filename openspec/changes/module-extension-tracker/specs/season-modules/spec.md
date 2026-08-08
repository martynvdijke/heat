## ADDED Requirements

### Requirement: Season module configuration
The system SHALL allow each season to declare which gameplay modules it plays with, stored in a `season_modules` join table (`season_id`, `module_id`, primary key on both). `POST /api/seasons` SHALL accept an optional `modules` array of module ids and persist the configuration. `GET /api/seasons` SHALL return the enabled module ids for each season.

#### Scenario: Season created with modules
- **WHEN** an admin POSTs to `/api/seasons` with a name and `modules: [3, 4]`
- **THEN** the season is created and rows are inserted into `season_modules` for modules 3 and 4

#### Scenario: Season created without modules
- **WHEN** an admin POSTs to `/api/seasons` with only a name (no `modules` field)
- **THEN** the season is created with no enabled modules (backward compatible)

#### Scenario: Season returns its configured modules
- **WHEN** an admin GETs `/api/seasons`
- **THEN** each season in the response includes the ids of its enabled modules

### Requirement: Season module list endpoint
The system SHALL expose `GET /api/modules` for the admin Season UI to render module selection checkboxes. The response SHALL include module id, name, description, and extension name.

#### Scenario: Season form loads available modules
- **WHEN** the admin Season form is opened
- **THEN** it fetches `/api/modules` and renders one checkbox per module

### Requirement: Season module display in admin
The admin Season UI SHALL show which modules a season has enabled (e.g. checkboxes reflecting the season's configured modules, and the enabled modules shown alongside the season row).

#### Scenario: Admin edits a season's modules
- **WHEN** an admin opens the season form for an existing season that has modules 2 and 5 enabled
- **THEN** the checkboxes for modules 2 and 5 are pre-checked

#### Scenario: Admin saves a season's module selection
- **WHEN** an admin submits the season form with a changed module selection
- **THEN** the `season_modules` rows are replaced to match the new selection

## REMOVED Requirements

_None_
