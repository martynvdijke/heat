## Why

The public stats page has a season dropdown, but it only filters two data sources (`/api/racer-stats` and `/api/rounds`); every "Performance Analysis" section, the track stats, ELO, streaks, and consistency ratings are always all-time. There is no way to view one statistic across multiple seasons, "All Time" is an unlabeled accident (empty selection) rather than a first-class option, and `/api/racer-stats?season_id=X` silently returns all-time data when a season has no final rounds — which hides missing data. Visitors cannot answer basic questions like "how did each driver do last season vs this season?" or "what are the career totals?"

## What Changes

- **Season scope control** on the public stats page: a multi-select season picker with an explicit **All Seasons** option (default) that scopes *every* section — top stat cards, charts, driver/track tables, and Performance Analysis — to the selection.
- **Cross-season comparison**: when 2+ seasons are selected, render a per-racer side-by-side comparison table (races, wins, podiums, points, spins, overheated per season) plus a grouped comparison chart for points/wins.
- **All-time aggregation**: "All Seasons" computes totals across every season from `round_snapshots`/`round_snapshot_scores`, clearly labeled as all-time.
- **Backend season-aware stats**: extend `season_id` filtering (and multi-season `season_ids` + all-seasons mode) to the remaining stats endpoints (`track-stats`, `track-performance`, `qualifying-delta`, `consistency`, `pace-heatmap`, `streaks`, `elo`, `head-to-head`, `race-incidents`, `points-progression`, `export`), where the underlying data can be season-scoped.
- **Fix the silent fallback**: `GET /api/racer-stats?season_id=X` must return an empty result (not all-time data) when the season has no final rounds, so the UI can show a truthful "No stats for this season" state.
- No changes to season creation/archival or round snapshotting.

## Capabilities

### New Capabilities
- `public-stats-season-scope`: The public stats page contract for scoping statistics by season — single-season, multi-season (comparison), and all-seasons (all-time) modes, including which sections must respect the scope and how empty seasons are surfaced.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `handlers/stats_basic.go`: remove silent all-time fallback; support `season_ids` multi-select and all-seasons aggregation.
- `handlers/stats_performance.go`, `handlers/stats_advanced.go`, `handlers/stats_incidents.go`: accept `season_id`/`season_ids` where data is season-scopeable.
- `racing/stats.go` (+ `racing/consistency.go`, `racing/elo.go`, `racing/streaks.go`, `racing/qualifying.go`, `racing/head_to_head.go`, `racing/track_stats.go` as needed): season-scoped query variants; `AllSeasonsStats` aggregation helper.
- `ts/stats.ts`, `static/stats.html`: multi-select season control, "All Seasons" option, comparison table/chart rendering, pass scope to all fetches (incl. `loadDeeperStats`).
- `main.go`: no new routes expected (existing endpoints gain params); update swagger annotations.
- Tests: `05_test_stats_test.go`, `14_test_admin_season_stats_test.go`, Playwright stats tests (`tests/`) updated for new scope behavior; new Go tests for season aggregation and empty-season behavior.
- Note: `race_history`/`race_results`/`race_events`/`lap_records` have no season column — season scoping for those endpoints requires joining through `round_snapshots` (by race name/date) or is left all-time; design decides per endpoint.
