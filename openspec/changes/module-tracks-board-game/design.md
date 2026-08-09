## Context

The app is a Go + Gin + SQLite companion for the Heat board game. Persistence mixes Ent auto-migration (`db/init.go` → `srv.Ent.Schema.Create`) with raw-SQL backfill migrations and idempotent seeders. Handlers query via `server.DB` (raw SQL) and the Ent client (`server.Ent`) for a few entities — notably tracks (`handlers/tracks.go` uses the Ent client for `GetTracks`/`SaveTrack`/`DeleteTrack`).

The extension/module tracker (open change `module-extension-tracker`) added `extensions`, `modules`, `season_modules` tables and an `extension_id` column on `tracks`, `upgrade_cards`, `legend_abilities`. Tracks today belong only to an extension (`extension_id`, 0→1 normalized to Base Game). Modules own nothing. The race track selector (`#track-select` in admin and controller) lists all tracks with no curation.

## Goals / Non-Goals

**Goals:**
- Let tracks belong to a gameplay module (`module_id`) as well as an extension, with UI to set it.
- Provide a curated "board game tracks" list (admin setting) distinct from custom tracks.
- Race track selector shows board game tracks + custom tracks, clearly grouped.

**Non-Goals:**
- Per-season track lists (seasons already declare modules; track curation stays global for now).
- Importing real Heat board game track data beyond what is seeded.
- Changing how race history records tracks (track_id already stores the chosen id).

## Decisions

### 1. `tracks.module_id` — module ownership of tracks
Add `module_id INTEGER NOT NULL DEFAULT 0` to `tracks` (0 = no module). A track can be attributed to an extension (already exists) and optionally to a module.

**Consistency rule:** when a track is assigned to a module, its `extension_id` is derived from the module's `extension_id`. The track editor exposes both selects; saving a module also updates the extension to match the module's owner. This keeps the extension catalog consistent (a track's extension is always the extension that provides its module).

**Alternative considered:** a `module_tracks` join table for many-to-many. Rejected — a physical track belongs to exactly one module in the board game; a plain FK column matches the existing `extension_id` pattern and keeps queries simple.

### 2. `board_game_tracks` table — curated board game track list
New table `board_game_tracks (track_id TEXT PRIMARY KEY)`. The set of track ids the group plays with. `SeedBoardGameTracks()` marks the 5 seeded base tracks as board game tracks on first run (idempotent: only when the table is empty). New tracks created afterwards are NOT board game tracks by default — the admin adds them via the editor.

**API:** `GET /api/board-game/tracks` returns `{track_ids: [...]}`; `PUT /api/board-game/tracks` replaces the whole set (full-replace semantics, simple and predictable).

**Alternative considered:** a `is_board_game` boolean column on tracks. Rejected — the board game list is a group-level setting, not a track attribute; a separate table lets the admin toggle membership without mutating track rows and keeps the "custom" notion trivially derivable (tracks not in the set).

### 3. Track selector grouping via `is_board_game` in `GetTracks`
`GET /api/tracks` gains `is_board_game` per track (computed by joining against `board_game_tracks`). The admin and controller selectors build two optgroups: **Board Game** (board game tracks, ordered by name) then **Custom** (the rest). No separate fetch needed; one API drives the dropdown.

### 4. Extension catalog shows module tracks
`GetExtensionDetail` returns modules with their own `tracks` arrays (from `tracks.module_id`), and the "Tracks" content table keeps its extension-level assignment dropdown. The module row in the detail UI lists the module's tracks.

### 5. Ent schema regeneration
`ent/schema/track.go` gains `module_id` (Int, default 0); regenerate with `go generate ./ent` (declared in `ent/generate.go`). `SaveTrack`/`GetTracks` map the new field. `GetTracks` still uses the Ent query; the board-game set is read from `h.S.DB` and applied to the mapped `models.Track`.

## Risks / Trade-offs

- [Ent regeneration churn] → Generated code is committed like the rest of `ent/`; run `go generate ./ent` and include the diff.
- [Full-replace PUT on board game tracks] → Admin UI loads current selection, so overwrite is intentional; document the semantics in the API.
- [Track with stale module after module deleted] → `DeleteModule` resets `tracks.module_id = 0` (and extension to the module's former owner's extension via the existing reset-to-base pattern).
- [Selector changes affect controller page too] → Both `admin.js` and `controller.js` share the same `/api/tracks` payload shape; grouping logic duplicated in each (small, explicit).
