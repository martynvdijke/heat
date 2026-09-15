## Why

The admin Season tab's Driver Statistics table and its edit modal do not show spins or overheated totals, even though the `racer_stats` table stores them and the stats API already returns them. The public stats page shows both numbers, so the admin view is incomplete and the data cannot be corrected from the admin UI.

## What Changes

- Add **Spins** and **Overheated** columns to the Driver Statistics table in the admin Season tab (Stats pane).
- Add **spins** and **overheated** number inputs to the add/edit driver-stats modal.
- `ts/admin.ts`: render spins/overheated in `renderStatsList()`, populate them in the stats modal load path, and include them in the save payloads of **both** stats save paths (stats modal and the inline stats save).
- Backend behavior is unchanged: `racer_stats` already has `spins`/`overheated` columns and `GET /api/racer-stats` already returns them; this change only surfaces them in the admin UI and ensures the admin save round-trips them (no data loss when saving from the modal).
- **Tests**:
  - Go handler test verifying `GET /api/racer-stats` returns `spins`/`overheated` and that the admin save endpoint persists them.
  - Playwright e2e verifying the Season tab Stats table shows Spins/Overheated columns and values, and that editing a driver's stats via the modal saves them.

## Capabilities

### New Capabilities
- `admin-season-stats`: The admin Season tab's driver statistics — table columns, modal form, and save behavior for a driver's race totals including spins and overheated.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `static/templates/tab-season.html`: Stats table headers, statsModal form inputs.
- `ts/admin.ts`: `renderStatsList()`, stats modal load/populate, both `saveStats` paths.
- Tests: new Go test file (e.g. `14_test_admin_season_stats_test.go`), Playwright updates to `tests/rounds-season-stats.spec.ts` or a new spec.
