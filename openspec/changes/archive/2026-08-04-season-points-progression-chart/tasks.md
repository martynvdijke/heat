## 1. Refactor `renderPointsChart` data preparation

- [x] 1.1 In `ts/stats.ts` inside `renderPointsChart(snapshots, allScores)`, replace the participation-driven `racerPoints[sc.racer_id].pts.push(sc.points)` logic with a round-index-aligned loop: for each snapshot index `i`, build/extend a per-driver `roundPts` array of length `snapshots.length` initialized to `0`, setting `roundPts[racer_id][i] = sc.points` when the driver has a score in snapshot `i`.
- [x] 1.2 Compute the cumulative array per driver from `roundPts` (running sum: `cum[i] = cum[i-1] + roundPts[i]`). Store both `roundPts` (raw, for the championship metric) and `cum` (for the chart data) on each driver's record.
- [x] 1.3 Keep the existing `labels = snapshots.map(s => s.race_name || 'R'+s.round)` unchanged.

## 2. Driver selection by final cumulative total

- [x] 2.1 Change the racer sort comparator from `a.pts[length-1]` (raw last-round points) to `a.cum[length-1]` (final cumulative total) descending.
- [x] 2.2 Add racer-name ascending as the tie-breaker so equal cumulative totals order deterministically.
- [x] 2.3 Keep the existing top-N selection count (5) and dataset construction loop, but set each dataset's `data` field to the driver's `cum` array instead of the raw `pts` array.

## 3. Preserve chart config and the `#championships` metric

- [x] 3.1 Leave the Chart.js `type: 'line'`, `fill`, `tension`, axis titles ("Points" Y, "Round" X), color palette, and canvas acquisition unchanged.
- [x] 3.2 Compute the `championships` count from the `roundPts` arrays (per-round raw points) rather than the cumulative arrays: for each round index, find the maximum raw points; every driver with that max increments their championship count. Set the `#championships` DOM text as before. (Equivalent semantics to the current code, retained per Decision 4.)
- [x] 3.3 Preserve the early-return guard for empty `snapshots`/`allScores` so the canvas stays blank on seasons with no rounds.

## 4. Build, verify, and lint

- [x] 4.1 Run `task build` (compiles TS via the existing build pipeline) and confirm `static/js/stats.js` is regenerated without errors.
- [x] 4.2 Run `task pre-push` (gofmt + Go tests + govulncheck + TS compile + go build) and confirm it passes. (No Go files change; this guards against accidental regressions.)
- [x] 4.3 Manual verification: load `/stats.html` for a season with multiple rounds, confirm the "Points Progression" chart shows monotonically non-decreasing lines, that a driver who missed a round still has a value at that round's X position (zero growth flat step), and that the displayed top-5 reflects the final championship standings rather than the last-round raw points.