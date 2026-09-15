## Context

The admin Season tab has three panes; the **Stats** pane shows a Driver Statistics table (Driver, Races, Gold, Silver, Bronze, Fastest Laps, DNF, DNS) and a statsModal for adding/editing a driver's season totals. Neither the table nor the modal includes **spins** or **overheated**, even though the backend fully supports both:

- `racer_stats` has `spins` and `overheated` columns.
- `racing.UpsertRacerStats` persists both columns on INSERT and UPDATE.
- `GET /api/racer-stats` returns both (`AllRacerStats`, `SingleRacerStatsFallback`, `RacerStatsBySeason`).
- `LockRound` increments `racer_stats.spins`/`overheated` on final lock.
- The public stats page already displays both columns.

The gap is frontend-only: `ts/admin.ts` `renderStatsList()` renders only dnf/dns badges, the stats modal only loads/saves dnf/dns, and both stats save paths send payloads without `spins`/`overheated`. Because `UpsertRacerStats` writes full rows, saving from the modal with the omitted fields silently resets `spins`/`overheated` to 0 — a data-loss bug this change fixes.

## Goals / Non-Goals

**Goals:**
- Show Spins and Overheated columns in the admin Season tab Stats table.
- Add spins/overheated inputs to the statsModal (add and edit paths).
- Round-trip spins/overheated through both stats save paths in `ts/admin.ts` so saving never zeroes them.
- Add Go handler tests and a Playwright e2e covering the new columns, the modal save, and the backend persistence.

**Non-Goals:**
- Any backend/API change — the API already stores and returns these fields.
- Changes to the public stats page.
- Aggregate or per-round admin views of spins/overheated beyond the per-driver totals.

## Decisions

### D1: No backend changes; fix the frontend round-trip
The backend is complete (`UpsertRacerStats` persists spins/overheated on INSERT and UPDATE; the read queries return them). The change touches only `static/templates/tab-season.html` and `ts/admin.ts`.

Rationale: the data flow is already correct; the admin UI was the only place dropping the fields.

### D2: Column placement and input style follow existing patterns
Spins and Overheated columns are added after DNS (matching the public stats page column order: Spins, Overheated). The modal gains two number inputs (`spins`, `overheated`, min=0) styled exactly like the existing `fastest_laps`/`dnf`/`dns` inputs in the statsModal form.

### D3: Both save paths include spins/overheated
Both `saveStats` paths (the stats modal save and the inline stats save) include `spins` and `overheated` in their payloads. To avoid drift, both payloads are built from the same in-memory stats object fields.

### D4: Tests
- New Go handler test file (next number after `13_test_tracks_modules_test.go`, i.e. `14_test_admin_season_stats_test.go`): create a racer, POST `/api/racer-stats` with spins/overheated, assert `GET /api/racer-stats` returns them and the DB row persists them.
- Playwright: extend `tests/rounds-season-stats.spec.ts` (or add a new spec) asserting the Season tab Stats table has Spins/Overheated columns with the driver's values, and that editing via the modal persists and re-renders them.

## Risks / Trade-offs

- [Modal save zeroing spins/overheated before this change] → Fixed by D3; the Go test asserts persistence explicitly.
- [Two save paths drifting] → Both paths read from the same stats fields; the Playwright test covers the modal path end-to-end.
- [Column-width crowding in the Stats table] → Existing table already has 8 columns with Bootstrap; two more numeric columns are narrow and match the public page.
