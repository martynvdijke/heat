## 1. Database layer

- [x] 1.1 Add `extension_id` field (Int, default 0) to `Track`, `UpgradeCard`, and `LegendAbility` schemas in `ent/schema/`
- [x] 1.2 Add raw-SQL migrations to `db/init.go`: `CREATE TABLE IF NOT EXISTS extensions`, `CREATE TABLE IF NOT EXISTS modules`, `CREATE TABLE IF NOT EXISTS season_modules`, and `ALTER TABLE ... ADD COLUMN extension_id` for tracks, upgrade_cards, legend_abilities
- [x] 1.3 Add `db/seed_extensions.go` with idempotent `SeedExtensions()` (Base Game is_base=1, Heavy Rain) and `SeedModules()` (base modules + a Heavy Rain module); call both from `db/init.go`
- [x] 1.4 Update `SeedTracks`, `SeedUpgrades`, `SeedLegendAbilities` to link seeded content to the Base Game extension

## 2. Backend handlers

- [x] 2.1 Add `handlers/extensions.go`: `GetExtensions` (list with content counts), `CreateExtension`, `UpdateExtension`, `DeleteExtension` (reset content links to 1, cascade module/season_module cleanup)
- [x] 2.2 Add `GetExtensionDetail` (extension + its modules/tracks/upgrades/legends), `GetModules` (with extension name), `CreateModule`, `UpdateModule`, `DeleteModule` (clean up season_modules)
- [x] 2.3 Add `AssignContentExtension` handler (`PUT /api/content/extension`) accepting content type (track|upgrade|legend), content id, extension id
- [x] 2.4 Extend `CreateSeason` in `handlers/rounds.go` to accept optional `modules` array and insert `season_modules` rows
- [x] 2.5 Extend the season list endpoint to return each season's enabled `module_ids`
- [x] 2.6 Register all new routes in `main.go` under the admin group
- [x] 2.7 Add htmx route `GET /api/html/admin/extensions` rendering `tab-extensions.html` (follow existing tab pattern in `main.go`)

## 3. Frontend

- [x] 3.1 Add "Extensions" nav button to `static/templates/admin-header.html` (data-tab-id="extensions", hx-get /api/html/admin/extensions)
- [x] 3.2 Create `static/templates/tab-extensions.html`: extensions table (name, description, base badge, content counts), create/edit modal, per-extension content listing, modules management, content assignment
- [x] 3.3 Add extension selector (populated from `/api/extensions`) to the track edit form (`HtmxTracksEditForm` in `handlers/htmx.go` + its template)
- [x] 3.4 Add module checkboxes to the Season form (htmx `seasonNewFormTmpl` in `handlers/htmx.go`, loaded from `queryModules()`) and wire season create to send `modules`

## 4. Tests & verification

- [x] 4.1 Add Go tests for extensions/modules CRUD, content assignment, and season module config (mirror existing handler test style in `0*_test.go`)
- [x] 4.2 Verify seeding idempotency and base-game linking in tests
- [x] 4.3 Run `task pre-push` (gofmt, go test, vet + govulncheck, tsc, build) and fix any failures
