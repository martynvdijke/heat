## 1. Backend — Rewrite season stats to use round snapshots

- [x] 1.1 Rewrite `RacerStatsBySeason()` in `racing/stats.go` to aggregate from `round_snapshot_scores` joined with `round_snapshots` (filtered by `season_id` and `status = 'final'`), including spins and overheated in the SELECT
- [x] 1.2 Rewrite `SingleRacerStatsBySeason()` similarly, filtering by both `season_id` and `racer_id`
- [x] 1.3 Update `AllRacerStats()` in `racing/stats.go` to include `spins` and `overheated` in the SELECT query
- [x] 1.4 Update `SingleRacerStatsFallback()` to include `spins` and `overheated` in the SELECT query
- [x] 1.5 Update `UpsertRacerStats()` to include `spins` and `overheated` in INSERT/UPDATE statements

## 2. Backend — Update driver share endpoint

- [x] 2.1 Update `GetDriverStatsByToken()` in `handlers/driver_share.go` to include `spins` and `overheated` in the SELECT query

## 3. Frontend — Add spins and overheated to stats page

- [x] 3.1 Add "Spins" and "Overheated" column headers to the Driver Performance table in `static/templates/stats.html`
- [x] 3.2 Update `renderDriverStatsTable()` in `ts/stats.ts` to display spins and overheated values in each row

## 4. Tests/Validation

- [x] 4.1 Update existing stats test in `05_test_stats_test.go` to verify round-based aggregation returns correct spins and overheated
- [x] 4.2 Update `08_test_api_test.go` driver-stats test to verify spins and overheated are returned

## Verification notes (2026-08-06)

- All 10/10 tasks complete. 2.1: handlers/driver_share.go:93 SELECT includes `spins`, `overheated`. 3.1: static/templates/stats.html:95-96 `<th>Spins</th><th>Overheated</th>`. 3.2: ts/stats.ts:218-219 renders `${s.spins||0}` / `${s.overheated||0}`.
- 4.1: 05_test_stats_test.go:17-18,50-53,85-88 asserts spins=8/overheated=3 from round-based aggregation. 4.2: 08_test_api_test.go:1087-1095 asserts spins/overheated fields in driver-stats API.
- `go test ./...` PASS. Committed in 5925692.
