## 1. Remove dead sections from stats.html

- [x] 1.1 Remove the 7 never-populated deeper-stats cards: Most Golds (`#most-golds-body`), Most Silvers (`#most-silvers-body`), Fastest Laps (`#fastest-laps-body`), Most Poles (`#most-poles-body`), Track Performance (`#track-performance-body`), Points Progression (`#points-progression-body`), Streaks (`#streaks-body`)
- [x] 1.2 Remove the Avg Lap Time column (th + td) from the Track Statistics table in `static/templates/stats.html`
- [x] 1.3 Remove the Lap Time Trends card (with `#laptime-chart` canvas) from `static/templates/stats.html`

## 2. Clean up ts/stats.ts

- [x] 2.1 Drop the avg-lap-time cell rendering from `renderTrackStatsTable` (~line 247)
- [x] 2.2 Remove references to the removed bodies from `loadDeeperStats`; keep only qualifying-delta, consistency, incidents, pace-heatmap fetches
- [x] 2.3 Fix empty-state placeholder colspans: Track Statistics table 4 → 3 columns; correct Driver Performance placeholder colspan to the real column count (8)

## 3. Tests

- [x] 3.1 Check `tests/page-layout.spec.ts` (and other specs) for assertions on removed elements; update any that reference them
- [x] 3.2 Add Playwright assertions that removed sections/columns are absent from the DOM (e.g., in a stats or page-layout spec)
- [x] 3.3 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
