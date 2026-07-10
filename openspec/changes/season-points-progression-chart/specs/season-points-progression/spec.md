## ADDED Requirements

### Requirement: Cumulative points progression chart
The season statistics page SHALL render a line chart in `#points-chart` where each line represents one driver and the Y value at each X position is that driver's **cumulative** points total from the start of the season through that round (inclusive).

#### Scenario: All rounds played
- **WHEN** a season has 3 rounds and a driver scores `10`, `25`, `18` in rounds 1, 2, 3 respectively
- **THEN** the driver's line data array is `[10, 35, 53]`

#### Scenario: Driver misses a round
- **WHEN** a season has 3 rounds; a driver plays round 1 (scores 10), misses round 2 (no `RoundSnapshotScore`), and plays round 3 (scores 18)
- **THEN** the driver's cumulative data array is `[10, 10, 28]` — the missed round contributes 0 to the cumulative sum and does not shift subsequent rounds left

#### Scenario: X axis represents rounds in season order
- **WHEN** the chart is rendered with snapshots ordered round ASC
- **THEN** each X-axis label is the snapshot's `race_name` (or `'R'+round` when `race_name` is empty), and array index `i` of every driver's data corresponds to snapshot `i` in the snapshots array

### Requirement: Round-index alignment for missed rounds
The chart data SHALL be index-aligned to the snapshots array. A driver absent from a given round's `scores` SHALL have `0` for that round's slot in the per-round-points array, and the cumulative sum SHALL treat that slot as `0`.

#### Scenario: Driver participates in non-consecutive rounds
- **WHEN** a season has 5 rounds; a driver participates only in rounds 1 and 5 (scoring 15 and 12)
- **THEN** the per-round-points array is `[15, 0, 0, 0, 12]` and the cumulative data array is `[15, 15, 15, 15, 27]`

#### Scenario: Multiple drivers in the same round
- **WHEN** round 2 of a season has scores for drivers A and B but not driver C
- **THEN** index 1 of driver A's and driver B's data reflects their round-2 points, and index 1 of driver C's per-round array is `0`

### Requirement: Driver selection by final cumulative total
When the chart limits the displayed drivers to a top-N set, the selection SHALL rank drivers by their final cumulative value (the value at the last index of their cumulative data array) descending, with racer name as a deterministic tie-breaker.

#### Scenario: Champion ranked above late-round winner
- **WHEN** driver A ends with cumulative total `120` (steady across rounds) and driver B ends with cumulative total `60` (boosted by a single high-scoring final round)
- **THEN** driver A ranks above driver B in selection order, regardless of their last-round raw points

#### Scenario: Tie on final cumulative total
- **WHEN** two drivers both end with cumulative total `90`
- **THEN** the driver whose name sorts first lexicographically ranks first, producing a deterministic ordering

### Requirement: Preserve championship count metric semantics
The `#championships` metric computed alongside the chart SHALL continue to count, per driver, the number of rounds in which that driver had the **highest raw per-round points** among all drivers in that round. It SHALL NOT be redefined in terms of cumulative totals.

#### Scenario: Championship count uses per-round raw points
- **WHEN** driver A leads raw points in 4 of 7 rounds but is overtaken on cumulative total late in the season
- **THEN** the driver's championship count contribution is `4`

#### Scenario: Tied top raw points in a round
- **WHEN** two drivers tie for the highest raw points in a round
- **THEN** each of those drivers is counted as holding the top for that round (i.e., both increment their championship count by 1 for that round) — consistent with the existing implementation's behavior

### Requirement: No backend or API changes
This change SHALL NOT modify any Go handler, route, model, database schema, or migration. It SHALL operate exclusively on the existing JSON shapes returned by `/api/rounds` and `/api/rounds/batch`.

#### Scenario: Existing API responses remain valid inputs
- **WHEN** the renderer is given an array of `RoundSnapshot` and a matching array of `RoundSnapshot`-with-`scores` objects whose `scores` arrays may omit a given `racer_id` for some rounds
- **THEN** the chart renders cumulative lines for the participating drivers without any new API field or backend round-trip