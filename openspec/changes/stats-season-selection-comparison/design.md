## Context

The public stats page (`static/stats.html`, `ts/stats.ts`) already has a single season dropdown (`#stats-season-select`) wired to `switchSeason()` → `loadSeasonStats(seasonId)`. Today it only scopes `/api/racer-stats?season_id=X` and `/api/rounds?season_id=X`; every other fetch is all-time. Key facts:

- **Two data families exist:**
  - **Snapshot-derived** (`round_snapshots` + `round_snapshot_scores`): per-season, finalized-round aggregates — used by `racer-stats?season_id=`, rounds, points chart, comparison data. `RacerStatsBySeason` (racing/stats.go:11) aggregates `position`, `points`, `dnf/dns`, `spins`, `overheated` per racer per season.
  - **Race-table-derived** (`race_results` JOIN `race_history`, plus `race_events` and `lap_records`): used by `track-stats`, `track-performance`, `qualifying-delta`, `consistency`, `pace-heatmap`, `streaks`, `elo`, `head-to-head`, `race-incidents`. `race_history` (ent/schema/race_history.go) has **no season column**: `(id, name, race_date, country, track, track_id, total_laps, race_type)`.
- `AllRacerStats` (racing/stats.go:143) reads the legacy `racer_stats` table (includes oneoff/manual entries) — not comparable to the snapshot-derived per-season numbers.
- `GET /api/racer-stats?season_id=X` silently falls back to `AllRacerStats` when the season has zero final rounds (handlers/stats_basic.go:39-41) — hides empty seasons.
- Seasons table: `seasons (id, name, start_date, end_date, status)` with `status IN ('active','archived')`. Rounds are created via `POST /api/rounds` (handlers/rounds.go) with `season_id`; race results are saved independently via `SaveRace` (handlers/race.go:118) which writes `race_history` with no season linkage. The only join key between them is `race_name`/`race_date`.
- Stats cache: `h.S.StatsCache` keyed by string (e.g. `stats:racer-stats:season:ID`); invalidated by prefix on stat updates.

## Goals / Non-Goals

**Goals:**
- A single season-scope control on the public stats page with three explicit modes: **All Seasons** (default, all-time), **one season**, **multiple seasons (comparison)**.
- Every stats section (top cards, charts, tables, Performance Analysis) respects the selected scope.
- Cross-season comparison: per-racer side-by-side table + grouped points/wins chart for the selected seasons.
- Truthful empty states: a season with no finalized rounds shows "no data", never silently substituted all-time numbers.
- Season-aware filtering on all stats endpoints whose data can be scoped.

**Non-Goals:**
- No changes to season CRUD/archival or round snapshotting.
- No new admin UI.
- No change to `racer_stats` legacy table semantics (remains the fallback only when no snapshot data exists for a racer).
- No ELO recalibration semantics beyond filtering which races are included.
- No per-race editing of season membership in v1.

## Decisions

### D1. API scope parameter: `season_ids` (comma-separated), absent = all seasons
All stats endpoints accept `season_ids=1,2,3`. Absent/empty = **all seasons (all-time)**. `season_id` remains accepted as an alias for a single value (backward compatible; existing Playwright/Go tests keep passing). The frontend always sends the resolved scope explicitly, so "All Seasons" is a first-class state, not an accident.

- Rationale: one param covers single, multi, and all-time uniformly; no URL-parsing special cases.
- Alternative rejected: `season_id` + separate `all=true` flag — two mechanisms for the same thing.

### D2. All-time ("All Seasons") is snapshot aggregation, not the `racer_stats` table
All-time racer stats = `RacerStatsBySeason` query with the `rs.season_id = ?` predicate removed (all `status='final'` snapshots). This makes All Seasons directly comparable with any single season (same source, same columns). The `racer_stats`-based `AllRacerStats` remains only as the fallback when a *racer* has no snapshot data (unchanged behavior for the `id=` path).

- Trade-off accepted: oneoff races and manually-entered `racer_stats` rows are excluded from All Seasons — correct, since "across all seasons" means season data.
- Cache key becomes `stats:racer-stats:seasons:<ids>` / `stats:racer-stats:seasons:all`.

### D3. Add nullable `season_id` to `race_history` to enable scoping of race-derived stats
The race-derived endpoints (track-stats, track-performance, qualifying-delta, consistency, pace-heatmap, streaks, elo, head-to-head, race-incidents) all JOIN `race_history`, which has no season link. Add `season_id INTEGER` (nullable, no FK constraint to keep migration simple) to `race_history`:

- **Migration** (db/init.go idempotent block, mirroring existing `CREATE INDEX IF NOT EXISTS` style): `ALTER TABLE race_history ADD COLUMN season_id INTEGER` guarded by a pragma table-info check; plus `CREATE INDEX IF NOT EXISTS idx_race_history_season ON race_history(season_id)`.
- **Backfill** (idempotent, runs on server start): for each season, `UPDATE race_history SET season_id = s.id FROM round_snapshots s WHERE s.season_id = ? AND s.race_name = race_history.name AND s.race_date = race_history.race_date` (SQLite-compatible per-table correlated update).
- **Write path**: `SaveRace` (handlers/race.go:118) accepts optional `season_id` in the request body (admin supplies it when recording a season race); when omitted and `race_type='season'`, fall back to resolving the active season at `race_date` (existing pattern in trmnl.go:101-114). Oneoff races get `NULL`.
- All race-derived queries gain an optional `AND rh.season_id IN (...)` filter derived from `season_ids`.

- Rationale: name+date matching (the only current link) is fragile (renames, duplicate dates); a column makes filtering one predicate and is cheap.
- Alternative rejected: leave race-derived sections all-time-only — violates the goal that the whole page respects scope.
- Alternative rejected: derive scope per request via `race_history` ↔ `round_snapshots` join on name+date — same fragility, computed on every request.

### D4. ELO and head-to-head scope by filtering included races
`ELORatings` and `HeadToHead` recompute over `race_results JOIN race_history`; with D3 they simply add the `season_ids` predicate. Ratings are order-dependent, so a scoped ELO is "ELO over these seasons' races" — documented, not a bug.

### D5. Frontend: one multi-select control drives all sections
Replace the single `<select>` with a multi-select control (`<select id="stats-season-select" multiple>` is Bootstrap-stylable but unwieldy; use instead a dropdown-panel with checkboxes, or keep it simple with a native multi-select — implementer picks, spec requires behavior):

- **All Seasons** pseudo-option at the top (checked by default).
- Checking "All Seasons" clears other selections; checking any season clears "All Seasons".
- 0 selected (after all unchecked) → treated as All Seasons.
- Exactly 1 selected → single-season view (today's behavior, but now applied to *all* sections including `loadDeeperStats`).
- 2+ selected → **comparison mode**: all sections aggregate across the selected seasons (sums), and a new **Season Comparison** card renders a per-racer × per-season table (races, wins, podiums, points, spins, overheated) plus a grouped bar/line chart of points or wins per season.
- URL state: `?seasons=1,2,3` query param so views are shareable/bookmarkable; the control reads it on load.
- Empty season handling: racer-stats returns `[]`; sections show the existing "No … yet" placeholders (already present), plus a banner when the whole season scope yields no data.

- Rationale: a single control avoids mode confusion between "select season" and "compare"; query-param state matches the existing stateless page model.
- Alternative rejected: separate "compare mode" toggle + second picker — two controls for one concept, more states to get wrong.

### D6. Deeper-stats fetches adopt the scope
`loadDeeperStats()` (ts/stats.ts:336) currently fires four unfiltered fetches. It becomes scope-aware: each of the four endpoints (qualifying-delta, consistency, incidents, pace-heatmap) accepts `season_ids`, and the fetches run with the current scope. Incidents additionally gain `season_ids` filtering via `race_events` → `race_history` join (D3 makes this a predicate on `rh.season_id`).

### D7. Fix the silent fallback
`GetRacerStats` (handlers/stats_basic.go:39-41): when a scope is explicitly provided (single or multi season) and yields no rows, return `[]` — never substitute `AllRacerStats`. The fallback remains only when the request is per-racer (`id=`) and no snapshot data exists.

## Risks / Trade-offs

- **`race_history.season_id` backfill may miss races** (renamed races, edited dates) → Mitigation: backfill is idempotent and runs every boot; races not matched simply stay un-scoped (excluded from single-season views, included in All Seasons). Document in admin; correct via future admin editing.
- **`ALTER TABLE` on SQLite locks the DB briefly** → Mitigation: tiny table, run at startup before serving; guarded idempotent block; matches existing migration style.
- **Behavior change: `?season_id=X` with no data now returns `[]` instead of all-time** → Mitigation: intended (D7); UI shows explicit empty states; existing tests asserting the old fallback get updated deliberately.
- **All Seasons no longer shows oneoff/manual `racer_stats` entries** → Mitigation: intentional (D2); the legacy path still serves the per-racer `id=` fallback.
- **Multi-select UX complexity / mobile** → Mitigation: keep the control compact (checkbox panel); spec only fixes behavior, layout handled in implementation; Playwright tests cover the three modes.
- **Scoped ELO/consistency results differ from historical all-time numbers** → Mitigation: correct by definition; no compatibility promise for computed ratings.

## Migration Plan

1. Ship D3 migration + backfill in the same release as the API changes (idempotent, safe to re-run).
2. Deploy order: backend first (endpoints accept `season_ids`; old single-select frontend keeps working via `season_id` alias), then frontend.
3. Rollback: revert frontend; endpoints retain `season_id` alias so the old page still functions. DB migration is additive and non-destructive.

## Open Questions

- Should "All Seasons" exclude archived-season data? (Default: include — all-time means all time.)
- Should the comparison table show only drivers present in all selected seasons, or union? (Default: union, sorted by total points.)
- Should `race_events`/`incidents` require race-type='season' when scoped? (Default: no — follow `race_history.season_id` only.)
