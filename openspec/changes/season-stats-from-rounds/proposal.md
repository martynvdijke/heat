## Why

The season statistics page currently derives driver points and standings from `race_results` (individual race finishes) rather than from `round_snapshot_scores` (official round snapshots). This means the stats page can show different numbers than what was officially recorded when a round was finalized. Additionally, `spins` and `overheated` data — which are already collected during round finalization — are not displayed anywhere on the stats page or in driver-share views, leaving a gap in driver performance visibility.

## What Changes

- **Season stats sourced from round snapshots**: The `/api/racer-stats?season_id=X` endpoint will aggregate points, positions, wins, podiums, fastest laps, DNF, DNS, spins, and overheated data from `round_snapshot_scores` (filtered to finalized rounds) instead of computing them from raw `race_results`. A fallback to the `racer_stats` table remains when no finalized rounds exist.
- **Spins and overheated added to driver stats table**: The frontend stats page table "Driver Performance" gains two new columns: Spins and Overheated.
- **Spins and overheated added to driver share page**: The `/api/shared/driver-stats` endpoint will include `spins` and `overheated` from the `racer_stats` table.
- **Race results still used for historical stats**: When no season is selected or no season/year filter is active, fall back to the existing `racer_stats` table (which is already updated on round finalization).

## Capabilities

### New Capabilities
- `round-based-season-stats`: Aggregate season driver statistics from finalized round snapshot scores, including spins and overheated.

### Modified Capabilities
*(None — no existing specs are being modified)*

## Impact

- **`racing/stats.go`**: `RacerStatsBySeason()` and `SingleRacerStatsBySeason()` to be rewritten to query `round_snapshot_scores` joined with `round_snapshots` (filtered by season_id and status='final') instead of `race_results`/`race_history`.
- **`racing/stats.go`**: `AllRacerStats()` and `SingleRacerStatsFallback()` — update SELECT queries to include `spins` and `overheated` columns.
- **`handlers/driver_share.go`**: `GetDriverStatsByToken()` — update SELECT to include `spins` and `overheated`.
- **`ts/stats.ts`**: Update `renderDriverStatsTable()` to display spins and overheated columns.
- **`static/templates/stats.html`**: Add "Spins" and "Overheated" column headers.
- **`models/models.go`**: No changes needed — `RacerStats` and `RoundSnapshotScore` already have the fields.
