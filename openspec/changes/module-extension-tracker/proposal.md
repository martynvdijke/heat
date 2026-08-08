## Why

The Heat: Pedal to the Metal companion app stores game content (tracks, upgrade cards, legend abilities) with no notion of which expansion pack or gameplay module it belongs to. Players own different combinations of the base game and expansions (e.g. Heavy Rain), and want to know "which extension has which track", "which upgrades came with Heavy Rain", and "which modules is this championship playing with". Today that information exists nowhere — content is a flat list and seasons have no module configuration.

## What Changes

- Introduce two new data entities, `extensions` and `modules`, to model the game's expansion packs and optional gameplay modules.
- Link existing content — tracks, upgrade cards, and legend abilities — to an extension (`extension_id`, defaulting to the base game). The model is open-ended: admins can add arbitrary extensions and attach content to them, so it supports content beyond Heavy Rain (custom expansions, future packs, house content).
- Seed the base game and Heavy Rain extension, plus a starter set of modules (Legends, Weather, Upgrades, Championship, etc.).
- Add an **Extensions** admin tab (a tracker): browse extensions, see which tracks / upgrades / legends / modules each one contains, and manage extensions, modules, and content assignments.
- Add **per-season module toggles**: a season (championship) declares which modules it plays with (e.g. "Legends + Weather"). Stored via a `season_modules` join table, editable from the Season UI, and shown in the tracker.
- Extend the existing content APIs (tracks, upgrade cards) to accept and return `extension_id`.

## Capabilities

### New Capabilities

- `extension-catalog`: Extension and module entities, content-to-extension linking (tracks, upgrade cards, legend abilities), seeding of base game + Heavy Rain content, and the admin catalog/tracker UI (list, detail, CRUD, assignment).
- `season-modules`: Per-season enabled module configuration (join table, API, season form UI, and display of configured modules in the admin).

### Modified Capabilities

<!-- No existing specs change -->

## Impact

- **DB**: new `extensions`, `modules`, `season_modules` tables (created via Ent schemas + raw SQL in `db/init.go`); `extension_id` column added to `tracks`, `upgrade_cards`, `legend_abilities` via the existing ALTER-TABLE migration pattern. Existing data defaults to base game.
- **Backend**: new handlers (extensions/modules CRUD, content assignment, season module config) in `handlers/`; route registration in `main.go`; new `SeedExtensions()` + `SeedModules()` in `db/`.
- **Frontend**: new `tab-extensions.html` htmx template + nav entry in `admin-header.html`; extension selector in track edit; module checkboxes in the season form (`tab-season.html`).
- **No breaking changes**: existing APIs remain compatible; new fields are optional with defaults.
