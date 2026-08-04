## Context

The season statistics page (`/stats.html`, served by template `static/templates/stats.html`, script built from `ts/stats.ts`) renders a Chart.js line chart in `<canvas id="points-chart">` titled "Points Progression". Today `renderPointsChart(snapshots, allScores)` (in `ts/stats.ts` around lines 78-143) builds one dataset per driver by pushing each `RoundSnapshotScore.points` value into an array as the snapshot loop visits that driver — producing per-round points, not a running total, and misaligning drivers who miss a round (their values are appended based on participation, not round index).

Data already available to the renderer:
- `snapshots`: array of `RoundSnapshot` ordered `round ASC` (from `GET /api/rounds?season_id=X`).
- `allScores`: result of `GET /api/rounds/batch?ids=...`, an array of snapshots each with `.scores: RoundSnapshotScore[]` (scores ordered by `position ASC`). Index-aligned with `snapshots`.

No backend or API changes are needed.

## Goals / Non-Goals

**Goals:**
- Plot cumulative points per driver across rounds in `#points-chart` (one line per driver).
- Data is round-aligned: a driver missing a round contributes `0` for that round's slot, so their line never shifts left.
- Rank/select drivers for display by the final cumulative total (last round's running sum), not by the raw points of the last round.
- Preserve existing visual conventions: line type, `fill`, `tension`, X/Y axis titles ("Round", "Points"), the rounding behavior, and the `#championships` metric computed alongside the chart.

**Non-Goals:**
- No backend, API, schema, or model changes.
- No changes to the separate placeholder `#points-progression-body` container (line 238 of the template) — it is a different (currently empty) widget.
- No changes to other charts on the page (`renderBattleChart`, `renderWinsChart`, etc.).
- No new HTTP dependencies (Chart.js already loaded from CDN).
- No attempt to fix the unrelated `fix-car-color-hex-rendering` concern here.

## Decisions

### Decision 1: Round-indexed zero-fill arrays, not participation-pushed arrays
Build, per driver, an array of length `snapshots.length` initialized to `0`. For round `i`, if that round's snapshot contains a score for the driver, set `cum[i] = sc.points`; otherwise leave `0`. Then compute a running sum in place: `for i>0: cum[i] += cum[i-1]`.

**Why over alternatives:** The current approach (`pts.push(sc.points)` per participation) breaks ordering when a driver misses a round — their points shift into a wrong X slot. Indexing by the round's position in the `snapshots` array (which is already `round ASC`) guarantees X alignment. Alternative considered: key by `s.round` literal and let Chart.js x-labeling sort — rejected because Chart.js line datasets require dense numeric-indexed `data` arrays; sparse data would render gaps rather than zero-fill, which is misleading (a missed round is a zero-point round, not a "no data" round).

### Decision 2: Selection/ranking by final cumulative total
Sort drivers by their last cumulative value (`cum[cum.length-1]`) descending and keep the existing top-N count (current code keeps 5). Tie-break by racer name for deterministic ordering.

**Why:** The current code sorts by `a.pts[length-1]` (raw last-round points) — a driver who won the final round but had few prior points would rank above the actual championship leader. Ranking by the final running sum reflects the actual championship standings, which is what a "progression" chart should convey.

### Decision 3: Running-sum transformation in place, no separate dataset per round
After zero-filling, mutate the per-driver `cum` array into a running tally before passing to Chart.js as `data`. This keeps one dataset per driver (same `data` shape as today) so no Chart.js config changes are needed.

**Why:** Avoids restructuring the dataset construction; only the data values change semantics from "raw per-round" to "cumulative". Alternative: use a custom Chart.js scriptable function to compute cumulative on the fly — rejected as needlessly indirect and harder to verify.

### Decision 4: Keep `#championships` metric semantics unchanged
The existing `championships` computation (count of rounds where a driver held the top score) continues to operate on **raw per-round points**, not cumulative. To support this without duplication, capture the raw per-round points in a parallel structure during the zero-fill pass, then use it for the championship count, and use `cum` only for the chart datasets.

**Why:** The "championships" metric semantically means "r rounds where this driver scored the most points" — a per-round concept. Re-defining it on cumulative totals would change the metric's meaning. Keeping both arrays (raw per-round for the metric, cumulative for the chart) preserves behavior and the `#championships` DOM element's contract.

### Decision 5: Labels unchanged
`labels = snapshots.map(s => s.race_name || 'R'+s.round)` stays as-is. This already orders by round ASC because `snapshots` is sorted.

**Why:** X-axis labels already encode round order; the only bug is in the data arrays (per-round vs cumulative, participation vs index-aligned). No label change required.

## Risks / Trade-offs

- **Large fields of drivers make the chart unreadable**: already mitigated by the existing top-N selection (preserved).
- **Drivers with identical names across data sources**: identifier is `racer_id` today; unchanged.
- **Empty season (no snapshots)**: existing guard returns early (canvas stays blank); preserved.
- **Cumulative lines are monotonic non-decreasing** → viewers could misread tails as "no growth" in late rounds. Acceptable trade-off; matches the standard championship-graph convention.