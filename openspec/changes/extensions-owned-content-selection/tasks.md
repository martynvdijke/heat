## 1. Database: owned_extensions table

- [x] 1.1 Add `owned_extensions(extension_id INTEGER PRIMARY KEY)` table creation + Base Game (is_base=1) owned seed to `db/init.go` (idempotent, `INSERT OR IGNORE`)

## 2. Backend: full content enumeration in extension catalog

- [x] 2.1 Expand `GetExtensionDetail` in `handlers/extensions.go` to return full `[]models.Track` (incl. geojson, length, module_id, is_board_game), `[]models.UpgradeCard` (incl. description, effects, card_type, cost) and `[]models.LegendAbility` (incl. description, ability_type, racer_name) per extension, replacing the truncated id/name summaries
- [x] 2.2 Upgrade `queryModuleTracks` to return the full track shape in the module list

## 3. Backend: owned-extensions APIs

- [x] 3.1 Add `GET /api/extensions/owned` returning owned extension ids (always includes Base Game) in `handlers/extensions.go`
- [x] 3.2 Add `PUT /api/extensions/owned` full-replace handler (force-add Base Game, ignore unknown ids) in `handlers/extensions.go`
- [x] 3.3 Include an owned flag in `GetExtensions`/`ExtensionSummary` so the admin tab can render ownership

## 4. Backend: owned-content filtering on selection endpoints

- [x] 4.1 Add optional `owned=1` filter to `GetTracks` in `handlers/tracks.go` (predicate: `extension_id = 0 OR extension_id IN (SELECT extension_id FROM owned_extensions)`); default behavior unchanged
- [x] 4.2 Filter `GetAvailableUpgradesForRacer` in `handlers/game_mechanics.go` by owned extensions (upgrade deck-builder)
- [x] 4.3 Filter `GetLegendAbilities` in `handlers/game_mechanics.go` by owned extensions (legend assignment catalog); leave `GetRacerLegendAbilities` unfiltered so assigned legends stay visible

## 5. Frontend: owned controls on the extension tab

- [x] 5.1 Add an Owned checkbox column to the extension table in `static/templates/tab-extensions.html` (Base Game rendered checked + disabled)
- [x] 5.2 Wire checkbox toggles to `PUT /api/extensions/owned` (send full current set) and refresh the owned state
- [x] 5.3 Richer catalog display: render full content info (track country/geojson-optional, upgrade card_type/cost/effects, legend ability_type/racer_name) from the expanded detail endpoint

## 6. Frontend: owned-content filtering in selection UIs

- [x] 6.1 Update the race-day track select builder in `ts/admin.ts` (and `static/templates/tab-race-day.html` if needed) to use `GET /api/tracks?owned=1`
- [x] 6.2 Update the controller track select builder in `ts/controller.ts` (and `static/controller.html`) to use `GET /api/tracks?owned=1`

## 7. Tests

- [x] 7.1 Add Go handler tests (new numbered file, e.g. `14_test_extensions_owned_test.go`): owned list/set APIs, Base Game always owned, full-replace semantics, and owned-content filtering on `/api/tracks?owned=1`, `/api/available-upgrades`, `/api/legend-abilities`
- [x] 7.2 Add/update Playwright e2e: extension tab owned toggles and race-day track select excluding unowned tracks
- [x] 7.3 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
