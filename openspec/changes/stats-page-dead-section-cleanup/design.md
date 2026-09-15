## Context

The public stats page (`static/templates/stats.html`, rendered by `ts/stats.ts`) contains sections that are never populated, a column that always renders a placeholder, and a chart that is never drawn:

- 7 deeper-stats cards permanently showing "No data yet": Most Golds (`#most-golds-body`), Most Silvers (`#most-silvers-body`), Fastest Laps (`#fastest-laps-body`), Most Poles (`#most-poles-body`), Track Performance (`#track-performance-body`), Points Progression (`#points-progression-body`), Streaks (`#streaks-body`).
- The **Avg Lap Time** column in the Track Statistics table, always rendered as `--:--` by `renderTrackStatsTable` (~line 247 of `ts/stats.ts`).
- The **Lap Time Trends** chart card (`#laptime-chart` canvas) — no rendering code exists anywhere.

The loader (`loadDeeperStats`) only populates four bodies: qualifying-delta, consistency, incidents, pace-heatmap. No API or data changes are involved; this is presentational cleanup per the user's request to "clean up the stats that are not in use".

## Goals / Non-Goals

**Goals:**
- Remove the 7 never-populated cards, the dead Avg Lap Time column, and the Lap Time Trends chart from `stats.html`.
- Remove the corresponding dead rendering from `ts/stats.ts` (avg-lap-time cell, references to removed bodies).
- Keep every working section untouched: top stat cards, Points Progression chart, Win Distribution, Driver Performance (incl. Spins/Overheated), Track Statistics (minus the dead column), Championship Battle, Points Leaderboard, ELO Ratings, Qualifying vs Race, Consistency, Race Incidents, Pace Heatmap.

**Non-Goals:**
- Backend/API changes of any kind.
- Implementing the removed features (no replacement analytics).
- Changes to the admin stats UI.

## Decisions

### D1: Remove dead HTML outright (user chose removal over hiding)
The 7 dead cards, the Avg Lap Time `<th>`/`<td>`, and the Lap Time Trends card are deleted from `static/templates/stats.html`. Hiding via CSS was considered and rejected — the user explicitly chose removal, and dead markup misleads.

### D2: Strip dead rendering from `ts/stats.ts`, keep `loadDeeperStats`
`renderTrackStatsTable` drops the avg-lap-time `<td>`; `loadDeeperStats` keeps only the four live fetches (qualifying-delta, consistency, incidents, pace-heatmap). No new code.

### D3: Fix placeholder colspans to match remaining columns
The empty-state placeholder rows must match the actual column count: the Track Statistics table goes from 4 to 3 columns (Track, Races, Winner), so its placeholder colspan drops 4 → 3. The Driver Performance table placeholder (colspan=6) is corrected to its real column count (8) while touching the same file.

### D4: Test surface
`tests/page-layout.spec.ts` is checked for assertions on removed elements and updated if needed; the Playwright coverage asserts removed sections are absent from the DOM.

## Risks / Trade-offs

- [A removed section is reintroduced later] → The new `public-stats-page` capability spec pins the exact live section set; a Playwright absence assertion guards it.
- [Colspan mismatch after removing a column] → Addressed by D3; verified visually via the existing Playwright layout checks.
- [User later wants one of the removed features] → Pure revert of static HTML/TS; no data was lost (the APIs never fed those sections).
