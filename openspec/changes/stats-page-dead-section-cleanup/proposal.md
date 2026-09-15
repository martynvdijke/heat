## Why

The public stats page renders several sections that are never populated (permanent "No data yet"), a Track Statistics column that always shows `--:--`, and a chart that is never drawn. These dead UI elements mislead visitors and add clutter; the working stats already cover the same ground.

## What Changes

- **Remove** the never-populated deeper-stats cards:
  - Most Golds (`#most-golds-body`)
  - Most Silvers (`#most-silvers-body`)
  - Fastest Laps (`#fastest-laps-body`)
  - Most Poles (`#most-poles-body`)
  - Track Performance (`#track-performance-body`)
  - Points Progression (`#points-progression-body`)
  - Streaks (`#streaks-body`)
- **Remove** the dead **Avg Lap Time** column from the Track Statistics table (always rendered as `--:--`).
- **Remove** the **Lap Time Trends** chart card (`#laptime-chart` canvas) — no rendering code exists for it.
- Clean up `ts/stats.ts`: drop references to removed sections/bodies; keep the working sections untouched (top stat cards, Points Progression chart, Win Distribution, Driver Performance incl. Spins/Overheated, Track Statistics, Championship Battle, Points Leaderboard, ELO Ratings, Qualifying vs Race, Consistency, Race Incidents, Pace Heatmap).
- Purely presentational cleanup — no API or data changes.

## Capabilities

### New Capabilities
- `public-stats-page`: The set of statistics sections the public stats page (`stats.html`) is required to display — serving as the contract that prevents dead placeholder sections from being reintroduced.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `static/templates/stats.html`: remove the 7 dead cards, the Avg Lap Time column, and the Lap Time Trends card.
- `ts/stats.ts`: remove `loadDeeperStats()` handling for removed bodies and the avg-lap-time rendering; keep the four live deeper-stats fetches (qualifying-delta, consistency, incidents, pace-heatmap).
- Tests: Playwright page-layout/stats checks updated if they assert on removed elements (`tests/page-layout.spec.ts`).
