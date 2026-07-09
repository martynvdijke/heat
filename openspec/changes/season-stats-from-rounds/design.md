## Context

The season statistics page (`/stats.html`) loads driver stats from `/api/racer-stats?season_id=X`, which currently queries `race_results` + `race_history` filtered by the season's date range. This approach has two problems:

1. **Stale data**: `race_results` records individual race finishes, but the official season standings should reflect the finalized round snapshots, which include adjusted scores, DNF/DNS flags, spins, and overheated data.
2. **Missing fields**: Spins and overheated data exist in `round_snapshot_scores` (set during round editing/finalization) and are propagated to `racer_stats`, but neither the season stats query nor the driver share endpoint includes them in their SELECT.

Round finalization (`FinalizeRound`) already updates `racer_stats` with aggregated spins and overheated counts. The `RoundSnapshotScore` schema already includes `spins` and `overheated` fields. The `RacerStats` model already has the fields. What's missing is the query layer and frontend display.

## Goals / Non-Goals

**Goals:**
- Season-filtered `/api/racer-stats?season_id=X` returns stats aggregated from finalized `round_snapshot_scores` for that season
- `spins` and `overheated` columns appear in the Driver Performance table on the stats page
- Driver share page (`/api/shared/driver-stats`) returns `spins` and `overheated`
- Fallback to `racer_stats` when no finalized rounds exist for the selected season
- All-rounds (no season filter) and single-racer lookups continue working with spins/overheated included

**Non-Goals:**
- Backfilling historical spins/overheated for rounds finalized before the columns existed (they default to 0)
- Changing the round finalization flow or email notifications
- Redesigning the stats page layout or adding new chart types
- Constructor/team standings changes

## Decisions

### 1. Source season stats from round_snapshot_scores instead of race_results

**Chosen approach:** Rewrite `racing.RacerStatsBySeason()` to run an aggregation query against `round_snapshot_scores` joined with `round_snapshots`, filtered by `season_id` and `status = 'final'`.

Query shape:
```sql
SELECT rss.racer_id,
       COUNT(*) as races,
       SUM(CASE WHEN rss.position = 1 THEN 1 ELSE 0 END) as wins,
       SUM(CASE WHEN rss.position = 1 THEN 1 ELSE 0 END) as gold,
       SUM(CASE WHEN rss.position = 2 THEN 1 ELSE 0 END) as silver,
       SUM(CASE WHEN rss.position = 3 THEN 1 ELSE 0 END) as bronze,
       0 as fastest_laps,           -- not stored in round_snapshot_scores
       SUM(rss.points) as points,
       SUM(CASE WHEN rss.dnf = 1 THEN 1 ELSE 0 END) as dnf,
       SUM(CASE WHEN rss.dns = 1 THEN 1 ELSE 0 END) as dns,
       SUM(rss.spins) as spins,
       SUM(rss.overheated) as overheated
FROM round_snapshot_scores rss
JOIN round_snapshots rs ON rs.id = rss.snapshot_id
WHERE rs.season_id = ? AND rs.status = 'final'
GROUP BY rss.racer_id
ORDER BY points DESC
```

**Alternatives considered:**
- *Keep querying race_results but add spins/overheated*: Spins and overheated don't exist in `race_results`, so we'd need a separate join or subquery against `round_snapshot_scores`. This couples two data sources and is more fragile.
- *Query racer_stats directly*: `racer_stats` already has aggregated data after finalization, but it doesn't record per-round data (no fastest_laps per round) and combining multiple seasons would be impossible.

**Rationale:** Round snapshots are the authoritative source for what happened each race weekend. They carry adjusted scores, the correct position, DNF/DNS flags, spins, and overheated. Using them as the source eliminates discrepancies between the stats page and the admin's round records.

### 2. Fastest laps not included from round snapshots

`RoundSnapshotScore` does not store `fastest_lap`. The current `RacerStatsBySeason()` queries it from `race_results.fastest_lap`. For now, fastest_laps will be set to 0 in the round-based aggregation. The stats page already handles this gracefully (shows 0). A future change could add `fastest_lap` to `round_snapshot_scores` if needed.

### 3. Fallback strategy

When a season has no finalized rounds (e.g., a brand new season), `RacerStatsBySeason()` currently falls back to `race_results` via the date-range approach. This fallback will be replaced with a simpler fallback: query the `racer_stats` table (which already contains accumulated data across all time). This matches the existing behavior for the "no season selected" path.

### 4. Update driver share query

The `GetDriverStatsByToken` query in `handlers/driver_share.go` selects from `racer_stats` but omits `spins` and `overheated`. The SELECT will be updated to include them.

## Risks / Trade-offs

- **[Accuracy] Fastest laps not tracked in round snapshots**: The stats page will show 0 fastest laps for season-filtered views. However, the racer_stats table still tracks them cumulatively (set manually via the admin stats editor). The "all stats" view (no season filter) continues to show fastest laps from racer_stats. → Acceptable trade-off since fastest laps are not yet part of the round workflow.
- **[Data gap] Rounds finalized before spins/overheated columns existed**: Those rounds' scores default to 0 for spins and overheated. → Acceptable — these are new tracking fields and there's no historical data to backfill.
- **[Consistency] The stats page mixes two data sources**: Season-filtered views come from round_snapshot_scores, unfiltered views from racer_stats. Aggregate numbers may differ slightly. → This is correct behavior: season data IS the round snapshot data.
