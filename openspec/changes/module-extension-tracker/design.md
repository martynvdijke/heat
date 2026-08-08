## Context

The app is a Go + Gin + SQLite companion for the Heat board game. Persistence is a mix: `db/init.go` runs Ent auto-migration (`srv.Ent.Schema.Create`) to create tables from `ent/schema/*.go`, then runs raw-SQL backfill migrations (`ALTER TABLE ... ADD COLUMN`, `CREATE TABLE IF NOT EXISTS`) and idempotent seeders (`SeedTracks`, `SeedUpgrades`, `SeedLegendAbilities`, ...). All handlers query the DB with raw SQL via `server.DB`; the Ent client is effectively vestigial for most entities. No Ent edges are used anywhere — relationships are flat foreign-key columns (`racer_id`, `track_id`).

Admin UI is htmx + Bootstrap, composed in `static/templates/admin.html` from tab templates (`tab-race-day`, `tab-season`, `tab-drivers`, `tab-config`) wired through the nav in `admin-header.html` and routes like `/api/html/admin/<tab>`.

Content entities to be tagged:
- `tracks` — id (string, e.g. `monza`), name, country, geojson, ... (5 seeded)
- `upgrade_cards` — name, description, card_type, cost, effects (8 seeded, `db/seed_game.go`)
- `legend_abilities` — name, description, ability_type, racer_name (5 seeded, `db/seed_game.go`)

`seasons` is managed with raw SQL (`handlers/rounds.go`); `CreateSeason` currently accepts only `{name}`.

## Goals / Non-Goals

**Goals:**
- Model expansion packs (`extensions`) and gameplay modules (`modules`) with an open-ended catalog — nothing hardcoded to Heavy Rain; admins can add arbitrary extensions and attach content to them.
- Link tracks, upgrade cards, and legend abilities to an extension (default = base game).
- Provide an admin **Extensions** tab that acts as the tracker: per-extension listing of modules/tracks/upgrades/legends, with CRUD and content assignment.
- Allow each season to declare which modules it plays with (`season_modules`), editable from the Season UI.
- Keep the change additive — no breaking API changes.

**Non-Goals:**
- Frontend gating of race-day panels by enabled modules (v1 records and displays configuration only; panel hiding is future work).
- Modeling per-race module configuration (configuration is season-scoped).
- Tagging `heat_cards` / `gear_shifts` / `weather_conditions` — these are runtime per-race/per-racer instances, not library content.
- New Ent schemas / Ent edge relations — new tables are raw SQL, matching `app_logs`/`log_settings` precedent and avoiding Ent codegen.

## Decisions

**D1. New tables are raw SQL, not Ent schemas.**
`extensions`, `modules`, `season_modules` created with `CREATE TABLE IF NOT EXISTS` in `db/init.go`, mirroring the existing `app_logs`/`log_settings` pattern. Rationale: avoids regenerating the Ent client (`go generate`), keeps the diff small, and matches how the codebase already handles auxiliary tables and all handler queries.
*Alternative considered:* Ent schemas + codegen — rejected as larger and riskier with no benefit since all queries are raw SQL anyway.

**D2. Content-to-extension linking via a flat `extension_id` column.**
`ALTER TABLE tracks|upgrade_cards|legend_abilities ADD COLUMN extension_id INTEGER NOT NULL DEFAULT 0` in `db/init.go`, following the existing unguarded ALTER pattern (errors discarded — the column is created by Ent on fresh DBs from the existing schema fields, and the ALTER is a no-op there). `0` means base game. No Ent edges — consistent with the codebase.
*Alternative considered:* Ent edges (many-to-many) — rejected; no edges exist in the codebase and raw SQL join queries are the established style.

**D3. Season modules stored in a `season_modules` join table.**
`PRIMARY KEY (season_id, module_id)` — correct for "which seasons use module X" tracker queries and avoids JSON/comma-string parsing. `POST /api/seasons` accepts an optional `modules` array; `GET /api/seasons` returns `module_ids` per season.
*Alternative considered:* comma-separated column on `seasons` — rejected; join table is cleaner for the tracker and not meaningfully more code.

**D4. New admin tab "Extensions" via htmx.**
Nav button in `admin-header.html` (`data-tab-id="extensions"`, `hx-get="/api/html/admin/extensions"`) + new `static/templates/tab-extensions.html` rendered by a route in `main.go`, matching the existing tab machinery (see `admin-footer.html` for the mount/load lifecycle). Content assignment (extension select) is also added to the track edit form surfaced by `HtmxTracksEditForm`.

**D5. Seeding is idempotent and extension-aware.**
New `SeedExtensions()` and `SeedModules()` (skip if rows exist). Insert `Base Game` (`is_base=1`) and `Heavy Rain` first, capture IDs, then seed base modules (Championship, Legend Drivers, Weather, Upgrades, Turbo) plus a Heavy Rain module (Extreme Weather). Existing seeders (`SeedTracks`, `SeedUpgrades`, `SeedLegendAbilities`) gain an `extension_id` = Base Game. No invented upgrade content — the UI is the mechanism for adding Heavy Rain or custom upgrades ("track more upgrades than Heavy Rain" is satisfied by the open catalog).

**D6. Extension deletion nulls links rather than cascades.**
`DELETE /api/extensions` sets `extension_id = 0` on its content (back to base game) and deletes its modules plus `season_modules` rows. Protects against orphaned content.

## Risks / Trade-offs

- **Unguarded ALTER on existing DBs** → The codebase already relies on this pattern with discarded errors; Ent creates the column on fresh DBs, so the ALTER only matters for existing DBs where it succeeds. Verified pattern from `db/init.go:63-65`.
- **`extension_id = 0` sentinel ambiguity** → Documented as "base game" everywhere; the Extensions UI always shows a Base Game entry (`is_base=1`) so content displays correctly.
- **New admin tab lifecycle** → `admin-footer.html` hardcodes initial-subtab load logic per tab; the new tab must not break the mount logic. Mitigation: follow the exact structure of existing tabs and verify the tab loads in the running app / e2e test.
- **Season API change** → `POST /api/seasons` becomes backward-compatible (modules optional); existing callers (the "New Season" form) are updated in the same change.

## Migration Plan

1. Deploy code; on startup `db/init.go` creates the three new tables, adds `extension_id` columns, and seeds extensions/modules (idempotent).
2. Existing tracks/upgrades/legends are implicitly base-game content (`extension_id = 0`), displayed under the Base Game extension.
3. Rollback: revert code — new columns/tables are inert if unused; `CREATE TABLE IF NOT EXISTS`/ALTER are additive and safe to leave in place.
