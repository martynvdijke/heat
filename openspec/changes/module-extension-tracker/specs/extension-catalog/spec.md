## ADDED Requirements

### Requirement: Extension catalog management
The system SHALL maintain an `extensions` table (id, name, description, is_base, sort_order) and expose admin CRUD endpoints so that admins can add, rename, update, and remove expansion packs. The catalog MUST NOT be limited to a fixed set of expansions — admins SHALL be able to create arbitrary extensions and attach content to them.

#### Scenario: Admin creates a new extension
- **WHEN** an admin POSTs to `/api/extensions` with a name and description
- **THEN** the extension is persisted and returned with its id, `is_base` defaulting to 0

#### Scenario: Admin updates an extension
- **WHEN** an admin PUTs to `/api/extensions` with an existing id and changed fields
- **THEN** the extension row is updated with the new values

#### Scenario: Admin deletes an extension
- **WHEN** an admin DELETE `/api/extensions` with an existing id
- **THEN** the extension and its modules are removed, `season_modules` rows referencing its modules are removed, and any tracks/upgrades/legends linked to it are reset to `extension_id = 0`

### Requirement: Base game and Heavy Rain seeded
The system SHALL seed a `Base Game` extension (marked `is_base = 1`) and a `Heavy Rain` extension on first initialization, so the tracker is populated out of the box. Seeding MUST be idempotent (skip when the extensions table already has rows).

#### Scenario: First startup seeds extensions
- **WHEN** the app starts and the `extensions` table is empty
- **THEN** `Base Game` and `Heavy Rain` extensions are inserted, and existing content (tracks, upgrades, legend abilities) is linked to the Base Game extension

#### Scenario: Restart does not duplicate extensions
- **WHEN** the app restarts with existing extension rows
- **THEN** no additional extension rows are inserted

### Requirement: Module catalog management
The system SHALL maintain a `modules` table (id, name, description, extension_id, sort_order) and expose admin CRUD endpoints. Each module SHALL optionally belong to an extension (`extension_id = 0` meaning base game). Starter modules (Championship, Legend Drivers, Weather, Upgrades, Turbo, and a Heavy Rain module) SHALL be seeded idempotently.

#### Scenario: Admin creates a module
- **WHEN** an admin POSTs to `/api/modules` with name, description, extension_id, and sort_order
- **THEN** the module is persisted and returned with its id

#### Scenario: Admin lists modules with their extension
- **WHEN** an admin GETs `/api/modules`
- **THEN** the response includes all modules with their `extension_id` and the extension name

#### Scenario: Admin deletes a module
- **WHEN** an admin DELETE `/api/modules` with an existing id
- **THEN** the module is removed and `season_modules` rows referencing it are removed

### Requirement: Content linked to extensions
The system SHALL tag tracks, upgrade cards, and legend abilities with an `extension_id` column (0 = base game) so that each piece of content can be attributed to the expansion that provides it. Admins SHALL be able to assign or change the extension for a track, upgrade card, or legend ability.

#### Scenario: Existing content defaults to base game
- **WHEN** the database is migrated and content already exists without an extension
- **THEN** all existing tracks, upgrade cards, and legend abilities have `extension_id = 0`

#### Scenario: Admin assigns content to an extension
- **WHEN** an admin PUTs to `/api/content/extension` with a content type (`track`, `upgrade`, or `legend`), a content id, and an extension id
- **THEN** the content row's `extension_id` is updated to the given extension id

#### Scenario: Upgrade cards beyond Heavy Rain are supported
- **WHEN** an admin creates a new extension (e.g. a custom or future pack) and assigns upgrade cards to it
- **THEN** the tracker shows those upgrades under that extension, demonstrating the catalog is not limited to Heavy Rain

### Requirement: Admin extension tracker UI
The system SHALL provide an **Extensions** tab in the admin UI listing all extensions with their content counts (tracks, upgrades, legends, modules). Selecting an extension SHALL show the full list of its modules, tracks, upgrade cards, and legend abilities. The UI SHALL support creating/editing extensions and modules and assigning content to extensions.

#### Scenario: Admin opens the Extensions tab
- **WHEN** an admin navigates to the Extensions tab in the admin UI
- **THEN** a table of extensions is shown with name, description, base-game badge, and counts of tracks, upgrades, legends, and modules

#### Scenario: Admin views an extension's content
- **WHEN** an admin expands or opens an extension
- **THEN** its modules, tracks, upgrade cards, and legend abilities are listed

#### Scenario: Admin assigns content from the UI
- **WHEN** an admin changes the extension of a track in the track editor
- **THEN** the track's `extension_id` is persisted and the tracker reflects the change

## REMOVED Requirements

_None_
