## Why

The extension/module tracker models which expansion packs and gameplay modules the group owns, and every track is attributed to an extension. But gameplay modules themselves cannot own tracks (e.g. a "Heavy Rain circuits" module that ships with its own circuits), and there is no notion of which tracks actually belong to the board game — the race track selector lists every track with no distinction, so admins cannot curate the track list their group plays with.

## What Changes

- Add `module_id` to `tracks` so tracks can belong to a gameplay **module** (in addition to an extension). Modules therefore become first-class owners of race tracks, and the extension catalog shows each module's tracks.
- Add a **Board Game Tracks** setting where the admin curates which tracks are part of their board game (base game + owned expansions/modules). New tracks default to "custom" (not part of the board game) unless the admin adds them.
- The race track selector used when setting up a race presents the **board game tracks** together with **custom tracks** (tracks not in the board game list), so the dropdown reflects exactly what the group can race on.
- Track editor gains a **Module** selector; the tracks table and extension detail show module attribution and board-game/custom badges.

## Capabilities

### New Capabilities

- `module-tracks`: Tracks can be attributed to a gameplay module (`module_id`), surfaced in the track editor, the tracks table, and the extension catalog (modules listed with their tracks).
- `board-game-tracks`: A curated board game track list (settings + API + UI) and a race track selector that presents board game tracks alongside custom tracks.

### Modified Capabilities

<!-- No existing specs change -->

## Impact

- **DB**: `ALTER TABLE tracks ADD COLUMN module_id INTEGER NOT NULL DEFAULT 0`; new `board_game_tracks` table (track_id PK) via `db/init.go`; idempotent seeder marks the seeded base tracks as board game tracks.
- **Backend**: `models.Track.ModuleID`; `GetTracks`/`SaveTrack` handle `module_id`; new board game tracks endpoints (`GET`/`PUT /api/board-game/tracks`); extension detail returns each module's tracks; Ent schema `track.go` gains `module_id` (regenerate Ent client).
- **Frontend**: module dropdown in the track edit modal (`tab-race-day.html` + `admin.js`); Board Game Tracks editor card in the Tracks pane; grouped board-game/custom optgroups in `#track-select` (admin + controller); badges in the tracks table.
- **No breaking changes**: existing APIs remain compatible; new fields default to 0/empty.
