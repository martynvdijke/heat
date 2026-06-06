## 1. Fix Race Report Data Contract

- [x] 1.1 Update `RaceReportData` struct: rename `race_name` JSON tag to `name`, remove `LapRecords` and `RaceRadio` fields
- [x] 1.2 Add `country` and `total_laps` to the SQL query in `RaceReport()` and struct
- [x] 1.3 Fix `RaceResultEntry` struct: add proper `DNS` serialization (ensure `dns` JSON key is consistent)
- [x] 1.4 Update `static/race-report.html` JS: use correct field names (`race_name` → `name`, `did_not_start` → `dns`), remove `window.print()` auto-trigger
- [x] 1.5 Remove the dead `LapRecordEntry` and `RaceRadioEntry` types from `racing/racing.go` (if not used elsewhere)
- [x] 1.6 Update tests: fix `TestRaceReportPage` to validate actual data shape

## 2. Fix Streaks() Duplicate Query Bug

- [x] 2.1 In `racing.go:Streaks()`, fix the first `Scan()` call to capture `totalRaces` and `avgPosition` into proper variables
- [x] 2.2 Remove the redundant second SQL query (lines 365-373)

## 3. Split racing/racing.go into Domain Files

- [x] 3.1 Create `racing/stats.go` with `RacerStatsBySeason`, `SingleRacerStatsBySeason`, `RacerStatsFallback`, `AllRacerStats`, `UpsertRacerStats`, `RacerInfo`
- [x] 3.2 Create `racing/track_stats.go` with `TrackStatsResult`, `TrackStats`, `TrackPerformanceData`, `TrackPerformance`
- [x] 3.3 Create `racing/head_to_head.go` with `HeadToHeadData`, `HeadToHead`, `PointsProgressionData`, `PointsProgression`
- [x] 3.4 Create `racing/streaks.go` with `StreakData`, `positionEntry`, `calcStreak`, `Streaks`, `AllStreaks`
- [x] 3.5 Create `racing/elo.go` with `ELORatingData`, `ELORatings`
- [x] 3.6 Create `racing/qualifying.go` with `QualifyingRaceDeltaData`, `QualifyingRaceDelta`
- [x] 3.7 Create `racing/consistency.go` with `ConsistencyRatingData`, `ConsistencyRatings`
- [x] 3.8 Move `RaceReportData`, `RaceResultEntry`, `RaceReport` to a new `racing/race_report.go` (with the fixes from section 1)
- [x] 3.9 Move `ExportStatsCSVData`, `ExportStatsCSV` to `racing/csv_export.go`
- [x] 3.10 Remove all moved code from `racing/racing.go`, keep only `SeasonDates` and `UpsertRacerStats` if they remain
- [x] 3.11 Update imports in `racing/` files (all same package, no external import changes)
- [x] 3.12 Verify racing package compiles and tests pass

## 4. Split handlers/stats.go into Domain Files

- [x] 4.1 Create `handlers/stats_basic.go` with `GetRacerStats`, `UpdateRacerStats`, `GetTrackStats`
- [x] 4.2 Create `handlers/stats_advanced.go` with `GetHeadToHead`, `GetPointsProgression`, `GetStreaks`, `GetELORatings`
- [x] 4.3 Create `handlers/stats_performance.go` with `GetTrackPerformance`, `GetQualifyingRaceDelta`, `GetConsistencyRatings`, `GetPaceHeatmap`
- [x] 4.4 Create `handlers/stats_incidents.go` with `GetRaceIncidentsReport`
- [x] 4.5 Move fixed `GetRaceReport` to `handlers/stats_basic.go` or a dedicated `handlers/race_report.go`
- [x] 4.6 Remove moved code from `handlers/stats.go` or delete the file if empty
- [x] 4.7 Verify handlers package compiles and tests pass

## 5. Split middleware/middleware.go into Separate Files

- [x] 5.1 Create `middleware/security.go` with `SecurityHeaders()`
- [x] 5.2 Create `middleware/csrf.go` with `CSRFMiddleware()`
- [x] 5.3 Create `middleware/request_id.go` with `RequestIDMiddleware()`
- [x] 5.4 Create `middleware/rate_limit.go` with `RateLimitMiddleware()`
- [x] 5.5 Create `middleware/auth.go` with `AuthMiddleware()`
- [x] 5.6 Create `middleware/umami.go` with `UmamiMiddleware()` and `umamiResponseWriter`
- [x] 5.7 Remove all code from `middleware/middleware.go`
- [x] 5.8 Verify middleware package compiles

## 6. Split db/db.go into Separate Files

- [x] 6.1 Create `db/init.go` with `Init()` and `SetServer()`
- [x] 6.2 Create `db/backup.go` with `CreateBackup()` and `PruneBackups()`
- [x] 6.3 Create `db/seed.go` with `SeedData()`, `SeedTracks()`, `SeedQuotes()`, `SeedTeams()`, `SeedSeason()`
- [x] 6.4 Create `db/seed_settings.go` with `SeedNotificationSettings()`, `SeedAISettings()`, `SeedEmailSettings()`, `SeedUmamiSettings()`, `SeedBackupSettings()`
- [x] 6.5 Create `db/seed_game.go` with `SeedUpgrades()`, `SeedLegendAbilities()`, `SeedSectors()`
- [x] 6.6 Create `db/helpers.go` with `BoolToInt()` and `EscapeHTML()`
- [x] 6.7 Remove all code from `db/db.go`
- [x] 6.8 Verify db package compiles

## 7. Remove Duplicate Functions and Cleanup

- [x] 7.1 Check callers of `RacerStatsFallback()` — if none, remove it; if called, redirect to `AllRacerStats()`
- [x] 7.2 Check callers of `SingleRacerStatsFallback()` — if none, remove it; if called, redirect to `SingleRacerStatsBySeason()`
- [x] 7.3 Strip stdout OpenTelemetry tracing from `telemetry.go` (keep Prometheus metrics)
- [x] 7.4 Replace `db.EscapeHTML()` with `html.EscapeString()` from stdlib (check callers)
- [x] 7.5 Run `task pre-push` to verify everything compiles and tests pass
