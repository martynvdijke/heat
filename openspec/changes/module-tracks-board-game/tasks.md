## 1. Data Model & Migrations

- [x] 1.1 Add `module_id` field to `ent/schema/track.go` (Int, default 0) and regenerate Ent client (`go generate ./ent`)
- [x] 1.2 Add `ALTER TABLE tracks ADD COLUMN module_id INTEGER NOT NULL DEFAULT 0` migration in `db/init.go`
- [x] 1.3 Create `board_game_tracks (track_id TEXT PRIMARY KEY)` table in `db/init.go`
- [x] 1.4 Add idempotent `SeedBoardGameTracks()` marking the seeded base tracks, call it from `db/init.go`

## 2. Backend: Module Tracks

- [x] 2.1 Add `ModuleID` to `models.Track`
- [x] 2.2 Update `GetTracks`/`SaveTrack` in `handlers/tracks.go` to map and persist `module_id`
- [x] 2.3 Derive `extension_id` from the module's owning extension in `SaveTrack` when `module_id > 0`
- [x] 2.4 Update `GetExtensionDetail` in `handlers/extensions.go` to include each module's tracks
- [x] 2.5 Reset `tracks.module_id = 0` and `extension_id = 1` in `DeleteModule` for tracks owned by the deleted module

## 3. Backend: Board Game Tracks

- [x] 3.1 Add `IsBoardGame` to `models.Track` and populate it in `GetTracks` from `board_game_tracks`
- [x] 3.2 Add `GetBoardGameTracks` and `SetBoardGameTracks` handlers with routes in `main.go`

## 4. Frontend

- [x] 4.1 Add Module dropdown to the track edit modal in `tab-race-day.html` and wire it in `admin.js`
- [x] 4.2 Add Module dropdown to the htmx track form in `handlers/htmx.go`
- [x] 4.3 Add Board Game Tracks editor card to the Tracks pane in `tab-race-day.html`
- [x] 4.4 Group `#track-select` into Board Game / Custom optgroups in `admin.js` and `controller.js`
- [x] 4.5 Add Board Game / Custom badges to the tracks table in `admin.js`
- [x] 4.6 Show module tracks in the extension detail UI in `tab-extensions.html`

## 5. Tests

- [x] 5.1 Tests: save/read track with `module_id` and derived extension
- [x] 5.2 Tests: board game tracks seed, GET, PUT replace semantics, `is_board_game` in track list
- [x] 5.3 Tests: module deletion resets tracks to base game
- [x] 5.4 Run `task pre-push` (fmt, tests, vet, TS compile, build)
