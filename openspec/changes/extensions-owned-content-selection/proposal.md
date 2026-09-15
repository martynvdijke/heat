## Why

The extension system lets admins create extensions and assign tracks, upgrades and legends to them, but every content-selection surface (race-day track select, deck-builder upgrades, legend assignment) shows ALL content regardless of extension. The group wants to only see and select content from extensions they actually own.

## What Changes

- **Full content enumeration in the extension catalog**: `GET /api/extensions/detail` will return tracks, upgrades and legends using the same full shapes the rest of the site uses (`models.Track` with geojson/length/module_id/is_board_game, `models.UpgradeCard` with description/effects/card_type, `models.LegendAbility` with description/ability_type/racer_name) instead of the current truncated id/name-only summaries.
- **Owned-extensions concept**: new `owned_extensions` table recording which extensions the group owns. The Base Game extension is always owned and cannot be unowned. Admin UI (extension tab) gains controls to mark extensions as owned/not owned.
- **New APIs**:
  - `GET /api/extensions/owned` returns the list of owned extension ids (or owned extensions with content).
  - `PUT /api/extensions/owned` replaces the owned set (Base Game always included).
- **Owned-content filtering applied to selection surfaces**:
  - Race-day track select (admin tab + controller) only lists tracks whose extension is owned.
  - Upgrade deck-builder (`GET /api/available-upgrades`) only returns upgrades from owned extensions.
  - Legend assignment (`GET /api/legend-abilities`, `GET /api/racer-legend-abilities`) only returns legends from owned extensions.
- Non-breaking: all existing content remains intact; ownership only filters selection lists, never deletes data.

## Capabilities

### New Capabilities
- `extension-content-enumeration`: Full content info (tracks/upgrades/legends) available in the extension catalog, matching the shapes used elsewhere on the site.
- `owned-extensions`: Ownership model for extensions — `owned_extensions` storage, Base Game always owned, admin management, and the rules for which content is considered owned.
- `content-selection-filtering`: All content-selection surfaces (track select, upgrade deck-builder, legend assignment) restrict their options to content from owned extensions.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `db/init.go`: new `owned_extensions` table + seed of Base Game as owned.
- `handlers/extensions.go`: expand `GetExtensionDetail` to full content shapes; add owned-extensions list/set handlers; include owned flag in `GetExtensions`/`ExtensionSummary`.
- `handlers/tracks.go`: `GetTracks` gains optional owned-only filtering.
- `handlers/game_mechanics.go`: `GetAvailableUpgradesForRacer`, `GetLegendAbilities`, `GetRacerLegendAbilities` filter by owned extensions.
- Frontend: `static/templates/tab-extensions.html` (owned controls + richer catalog display), `static/templates/tab-race-day.html`, `static/controller.html`, `ts/admin.ts` (track-select builder), `ts/controller.ts` (track-select builder).
- Tests: Go handler tests (extensions/owned/content filtering) and Playwright e2e for the admin extension tab and race-day track select.
