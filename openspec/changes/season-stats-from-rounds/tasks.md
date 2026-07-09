## 1. Backend — Rewrite season stats to use round snapshots

- [x] 1.1 Rewrite `RacerStatsBySeason()` in `racing/stats.go` to aggregate from `round_snapshot_scores` joined with `round_snapshots` (filtered by `season_id` and `status = 'final'`), including spins and overheated in the SELECT
- [x] 1.2 Rewrite `SingleRacerStatsBySeason()` similarly, filtering by both `season_id` and `racer_id`
- [x] 1.3 Update `AllRacerStats()` in `racing/stats.go` to include `spins` and `overheated` in the SELECT query
- [x] 1.4 Update `SingleRacerStatsFallback()` to include `spins` and `overheated` in the SELECT query
- [x] 1.5 Update `UpsertRacerStats()` to include `spins` and `overheated` in INSERT/UPDATE statements

## 2. Backend — Update driver share endpoint

- [ ] 2.1 Update `GetDriverStatsByToken()` in `handlers/driver_share.go` to include `spins` and `overheated` in the SELECT query

## 3. Frontend — Add spins and overheated to stats page

- [ ] 3.1 Add "Spins" and "Overheated" column headers to the Driver Performance table in `static/templates/stats.html`
- [ ] 3.2 Update `renderDriverStatsTable()` in `ts/stats.ts` to display spins and overheated values in each row

## 4. Tests/Validation

- [ ] 4.1 Update existing stats test in `05_test_stats_test.go` to verify round-based aggregation returns correct spins and overheated
- [ ] 4.2 Update `08_test_api_test.go` driver-stats test to verify spins and overheated are returned
