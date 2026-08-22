## 1. Database migration & backfill

- [x] 1.1 Add idempotent `ALTER TABLE race_history ADD COLUMN season_id INTEGER` migration guarded by a pragma table-info check in `db/init.go`, plus `CREATE INDEX IF NOT EXISTS idx_race_history_season ON race_history(season_id)`
- [x] 1.2 Add idempotent startup backfill linking `race_history` rows to seasons by matching `round_snapshots` on `race_name`/`race_date` (correlated UPDATE per season; safe to re-run)
- [x] 1.3 Update `ent/schema/race_history.go` with the new `season_id` field and regenerate ent code (check `go generate`/ent workflow used by repo)
- [x] 1.4 Add Go test asserting the migration is idempotent and the backfill links matching race rows (test in `09_test_infrastructure_test.go` style)

## 2. Backend: season_ids scope on stats endpoints

- [x] 2.1 Add a shared scope-parsing helper (e.g. in `handlers/stats_*` or `racing/`) that parses `season_ids` (comma-separated ints) and `season_id` alias into a list; invalid values → HTTP 400
- [x] 2.2 `racing/stats.go`: generalize `RacerStatsBySeason` into `RacerStatsBySeasons(db, seasonIDs []int)` (empty slice = all seasons, no predicate); keep `RacerStatsBySeason` as a thin wrapper
- [x] 2.3 `handlers/stats_basic.go` `GetRacerStats`: use scope helper; **remove the silent all-time fallback** for explicit scopes (empty result when scope yields no rows); scope-aware cache keys (`stats:racer-stats:seasons:<ids>` / `:all`); keep `id=` per-racer fallback to `SingleRacerStatsFallback` only when no snapshot data exists
- [x] 2.4 `racing/track_stats.go` + `handlers/stats_basic.go` `GetTrackStats`/`GetTrackPerformance`: accept `season_ids` via `rh.season_id` predicate; scope-aware cache keys
- [x] 2.5 `racing/consistency.go` + `GetConsistencyRatings`: accept `season_ids` via `rh.season_id` predicate; scope-aware cache key
- [x] 2.6 `racing/qualifying.go` + `GetQualifyingRaceDelta`: accept `season_ids` predicate
- [x] 2.7 `racing/elo.go` + `GetELORatings`: accept `season_ids` predicate; scope-aware cache key
- [x] 2.8 `racing/streaks.go` + `GetStreaks` (all-racers mode): accept `season_ids` predicate; scope-aware cache key
- [x] 2.9 `racing/head_to_head.go` + `GetHeadToHead`: accept `season_ids` predicate
- [x] 2.10 `GetPaceHeatmap`: accept `season_ids` by joining `lap_records` → `race_history` on `season_id`
- [x] 2.11 `GetRaceIncidentsReport`: accept `season_ids` by joining `race_events` → `race_history` on `season_id`
- [x] 2.12 `handlers/race.go` `SaveRace`: accept optional `season_id` in request body; when absent and `race_type='season'`, resolve active season at `race_date` (reuse trmnl.go:101-114 pattern); write it to `race_history`
- [x] 2.13 Update swagger annotations for all changed endpoints (`season_ids`/`season_id` params)
- [x] 2.14 Update/extend Go tests in `05_test_stats_test.go` and `14_test_admin_season_stats_test.go` for: season_ids multi-scope, all-seasons aggregation, empty-season returns `[]`, invalid season_ids → 400

## 3. Frontend: season scope control

- [x] 3.1 Replace the single `<select id="stats-season-select">` in `static/stats.html` with a multi-season scope control (checkbox panel or native multi-select) including an "All Seasons" option; keep element id/behavior contract used by `ts/stats.ts`
- [x] 3.2 `ts/stats.ts`: rework `loadSeasonStats` to resolve scope from `?seasons=` URL param (`all` or comma-separated ids, default all) and to pass `season_ids` to every fetch (`/api/racer-stats`, `/api/rounds`, and the four deeper-stats endpoints)
- [x] 3.3 `ts/stats.ts`: make `loadDeeperStats` scope-aware (accept and forward `season_ids` to qualifying-delta, consistency, incidents, pace-heatmap)
- [x] 3.4 `ts/stats.ts`: implement comparison mode — when 2+ seasons selected, render per-racer × per-season table (races, wins, podiums, points, spins, overheated) and a grouped points/wins chart; add the comparison card markup to `static/stats.html`
- [x] 3.5 `ts/stats.ts`: update URL via `history.replaceState` on scope change (no reload) and re-render; enforce "All Seasons" exclusivity rules (checking All clears others; all unchecked → All)
- [x] 3.6 Render truthful empty states for empty scopes (existing "No … yet" placeholders + a banner when the whole scope has no data); ensure no all-time data leaks into an empty single-season view
- [x] 3.7 Rebuild TS → `static/js/stats.js` (repo build step, see `build.mjs`/Taskfile)

## 4. Tests & verification

- [x] 4.1 Update Playwright stats tests in `tests/` (e.g. `page-layout.spec.ts` / stats specs) for: All Seasons default, single-season scope, multi-season comparison table/chart, `?seasons=` deep links, empty-season empty states
- [x] 4.2 Run `task pre-push` (gofmt, `go test ./...`, vet + govulncheck, TS compile, Go build) and fix failures
- [x] 4.3 Manual smoke check: open `/stats.html?seasons=all`, switch to one season and to two seasons, verify all sections re-render and comparison view appears; verify a season with no final rounds shows empty states
