## Why

The controller page is the operator's single pane of glass, but three pieces of race context are missing or buried:

1. **Weather** is only visible inside the collapsible Conditions card (which can be closed) as a small line of text — the Race Control card shows no conditions at all.
2. **Standings rows** show position, name, and car, but no gap to the leader, even though lap data (`lap_records`) already records every driver's position per lap.
3. **Next race date** is stored in `race_info` (`next_race_date`, settable in admin and returned by `GET /api/race-info`) but is never shown on the controller.

All three are client-side gaps: every required value is already served by existing endpoints.

## What Changes

- **Weather chip** in the Race Control card header (always visible): condition icon + name, populated on page load from `GET /api/weather?race_id=0` and updated immediately after the operator sets weather.
- **Gap to leader** per driver row in the standings list, computed from recorded lap data (completed-laps difference): leader row shows "LEAD", trailing drivers show "+N" when behind by at least one lap, same-lap drivers show no gap. Column hidden entirely when no lap records exist yet.
- **Next race countdown** in the Configuration card: "Next race: 2026-08-22 · in 7 days" from `GET /api/race-info`; hidden when `next_race_date` is empty; refreshed on load and after saving race settings.
- **No backend changes**.
- **Tests**: Playwright e2e covering the three additions and their empty-data states.

## Capabilities

### New Capabilities
- `controller-polish`: Always-visible weather on the Race Control card, per-driver gap to leader in the standings list, and the next race date countdown on the controller page.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `static/controller.html`: weather chip markup in the Race Control card, gap column in the standings template, next-race line in the Configuration card.
- `ts/controller.ts`: weather fetch + chip update in `setWeather`; gap computation + rendering in `renderStandings()`; race-info fetch + countdown render.
- `static/style.css`: minor styles for the chip and gap column.
- Tests: Playwright (controller spec). No Go changes.
