## Why

The season statistics page's "Points Progression" chart plots each driver's per-round points as separate, disconnected per-round values rather than a cumulative running total. This hides the actual championship standings trajectory that viewers expect — they cannot see who was leading at any given round or how the points gap evolved over the season. The chart also misaligns rounds for drivers who miss a round (they "shift left" by participation rather than by round index).

Separately, car colors stored as hex in the database currently do not render on the home page. That issue is already covered by the existing apply-ready change `fix-car-color-hex-rendering` and is NOT duplicated here.

## What Changes

- Replace the per-round points data series in the "Points Progression" chart (`#points-chart` on `/stats.html`) with a **cumulative** running total per driver, aligned by round index (zero-filled for missed rounds).
- X axis = round number in season order (already supported by existing labels `race_name || 'R'+round`); one line per driver.
- Keep the existing top-N driver selection logic but rank by the **final cumulative** total (the last round's running sum) rather than the last round's raw points.
- Preserve the existing `#championships` metric computation triggered by the chart renderer.
- Frontend-only: modify `ts/stats.ts` `renderPointsChart`. No API or backend changes.

## Capabilities

### New Capabilities
- `season-points-progression`: Season statistics page capability for plotting each driver's cumulative points across rounds as a progression line chart, with round-aligned zero-fill for missed rounds and ranking by final cumulative total.

### Modified Capabilities
<!-- None — no existing spec-level requirements are changing. -->

## Impact

- **Code**: `ts/stats.ts` (function `renderPointsChart`, lines ~78-143). No backend, API, schema, or model changes.
- **Dependencies**: Chart.js (already loaded via CDN on `stats.html`). No new dependencies.
- **Compatibility**: Independent of and compatible with the existing apply-ready changes `fix-car-color-hex-rendering` and `season-stats-from-rounds`. The progression chart consumes the existing `/api/rounds` and `/api/rounds/batch` responses unchanged.
- **UX**: Viewers see cumulative championship trajectories instead of disconnected per-round points; missed rounds no longer shift a driver's line left.