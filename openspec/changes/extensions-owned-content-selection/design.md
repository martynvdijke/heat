## Context

The extension system (previous change) lets admins create extensions and assign tracks, upgrades and legends to them, but every content-selection surface shows ALL content regardless of extension:

- Race-day track select (admin tab + controller) — built from `GET /api/tracks`, grouped "Board Game"/"Custom".
- Upgrade deck-builder — `GET /api/available-upgrades?racer_id=` returns `card_type='upgrade'` cards not already in `player_upgrades`, with no extension filter.
- Legend assignment — `GET /api/legend-abilities` returns the full legend catalog.

The extension catalog endpoint (`GET /api/extensions/detail`) currently returns truncated summaries (id/name only) for tracks, upgrades and legends, so it cannot serve as the source of truth for content selection.

Existing precedent in the codebase: raw-SQL tables with flat FK columns, idempotent seeders, additive non-breaking changes, and a full-replace PUT for membership-style tables (`board_game_tracks`). The Base Game is the extension with `is_base=1` (id 1); content with `extension_id=0` is treated as Base Game content everywhere.

## Goals / Non-Goals

**Goals:**
- Enumerate full content (tracks/upgrades/legends) per extension in the catalog, using the same shapes the rest of the site uses (`models.Track`, `models.UpgradeCard`, `models.LegendAbility`).
- Introduce an owned-extensions concept: a small persistent set of extension ids the group owns, with Base Game always owned.
- Filter all content-selection surfaces to owned content, server-side, so unowned content never leaks into option lists.
- Provide admin controls to mark extensions owned/not owned.

**Non-Goals:**
- Deleting or hiding content in management surfaces: the extension catalog and the track/upgrade/legend management lists still show ALL content. Ownership only restricts *selection* lists.
- Per-user or per-season ownership (single global set for the whole group).
- Changing the board-game track list or module behavior.
- Migration of existing data: all existing content keeps its extension assignment.

## Decisions

### D1: `owned_extensions` membership table over a column on `extensions`
A new table `owned_extensions(extension_id INTEGER PRIMARY KEY)` records membership. Base Game (is_base=1) is seeded as owned and enforced in code.

Rationale: mirrors the existing `board_game_tracks` precedent (membership set + full-replace PUT), keeps the `extensions` catalog table purely descriptive, and makes the owned set trivially replaceable in one transaction.
Alternative considered: `is_owned` column on `extensions` — rejected because full-replace updates then need per-row writes, and Base Game enforcement is a special case that is cleaner in the membership handler.

### D2: Full content shapes in `GetExtensionDetail`, no new summary structs
The detail endpoint returns `[]models.Track` (incl. geojson, length, module_id, is_board_game), `[]models.UpgradeCard` (incl. description, effects, card_type, cost) and `[]models.LegendAbility` (incl. description, ability_type, racer_name) per extension.

Rationale: the frontend can then drive selection lists from the same data the rest of the site uses; no parallel truncated model to maintain. The existing per-module track list in the detail response is upgraded to the full track shape as well.
Alternative considered: keeping summary structs — rejected, it forces a second set of rendering code and drifts from the site shapes.

### D3: Server-side ownership filtering on selection endpoints
Filtering is applied in SQL at the API layer, not in the browser:

- `GET /api/tracks?owned=1` — optional param used by selection UIs (admin + controller track selects). Without the param, behavior is unchanged (management/catalog use full list).
- `GET /api/available-upgrades` — deck-builder option list filtered.
- `GET /api/legend-abilities` — legend assignment catalog filtered.

Ownership predicate: `extension_id = 0 OR extension_id IN (SELECT extension_id FROM owned_extensions)` — `extension_id=0` content is Base Game content (always owned), consistent with existing normalization.

Rationale: the server is authoritative; the client just renders. Avoids duplicating the filter across `ts/admin.ts` and `ts/controller.ts` and prevents unowned content from leaking even if a client forgets to filter.
Alternative considered: client-side filtering of already-fetched lists — rejected (duplicate logic, leaks unowned ids into the browser, easy to regress).

### D4: Already-assigned content is data, not an option list — not ownership-filtered
`GET /api/racer-legend-abilities` (legends currently assigned to a racer) and the purchased-upgrades display are NOT filtered. If an extension is unowned but a racer still has one of its legends assigned, the legend must remain visible so it can be unassigned.

This refines the proposal's mention of filtering `GetRacerLegendAbilities`: filtering the assignment *catalog* is correct; filtering the racer's *assigned* list would make assigned legends invisible and un-unassignable.

### D5: Admin controls as per-row Owned checkboxes on the extension tab
The extension tab gains an "Owned" checkbox column; toggling a checkbox immediately sends the full current owned set via `PUT /api/extensions/owned` (server adds Base Game back regardless). Base Game's checkbox is rendered checked and disabled.

Rationale: matches the existing per-row action pattern on the extension tab (`assignContent`), and the full-replace PUT keeps the server the single source of truth.
Alternative considered: a separate save-all button — rejected, inconsistent with the tab's immediate-action pattern.

### D6: Owned APIs
- `GET /api/extensions/owned` → `{owned_ids: []int}` (always includes Base Game).
- `PUT /api/extensions/owned` → full-replace `{owned_ids: []int}`; Base Game id is force-added; unknown ids ignored (same tolerance as `SetBoardGameTracks`).

## Risks / Trade-offs

- [Deck-builder silently shrinks when extensions are unowned] → Ownership changes are explicit admin actions; the extension tab shows counts and ownership side by side. Re-owning restores options immediately.
- [Filtering the racer legend list would break unassignment] → Mitigated by D4: only the catalog is filtered; assigned legends always render.
- [Full content payloads enlarge the detail response] → The catalog is only loaded on demand (one extension at a time); acceptable for an admin tab.
- [Race-day selects change for the group when ownership changes mid-season] → Ownership is a deliberate group-level setting; the design makes the owned set visible and editable in one place.
