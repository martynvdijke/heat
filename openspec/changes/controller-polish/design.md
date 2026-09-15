## Context

`ts/controller.ts` is polling + POST driven (no WebSocket). Relevant current behavior:

- `renderStandings()` renders each driver row from `controllerRacers` (position button, profile picture, name, car name, blue/black-white flag buttons, up/down buttons) — no gap information.
- `setWeather()` POSTs `/api/weather` with a hardcoded `lap_end: 999` and a `gripMap` (dry 1.0, damp 0.85, wet 0.7, torrential 0.5), then updates only the `#current-weather` text inside the Conditions card.
- `GET /api/race-info` returns `country, track, laps, track_id, next_race_date` (empty string when unset; YYYY-MM-DD format enforced server-side).
- `lap_records` (race_id, lap_number, racer_id, position, …) are written by "Record Lap Positions", which records every driver's position for the given lap. `GET /api/lap-records` returns them.

## Goals / Non-Goals

**Goals:**
- Weather always visible on the Race Control card.
- Lap-based gap to leader in the standings list, graceful when no lap data exists.
- Next race date countdown in the Configuration card, hidden when unset.
- Zero backend changes.

**Non-Goals:**
- Time-based gaps (lap times are not recorded anywhere — `lap_records` stores positions only).
- Server-side gap computation or new endpoints.
- Weather on other pages (covered by the separate weather change).
- Editing `next_race_date` from the controller (admin-only, as today).

## Decisions

### D1: Gap semantics from recorded lap data
A driver's completed laps = the highest `lap_number` in `lap_records` for that driver; the leader's completed laps = the highest `lap_number` where the leader appears (position 1). Gap = leader − driver. Render: leader row shows "LEAD"; drivers with gap ≥ 1 show "+N"; drivers on the same lap as the leader show nothing (F1 convention — on the same lap). If no lap records exist, the gap column is hidden entirely.

Data source: `GET /api/lap-records?race_id=0` fetched on load and after `recordCurrentLap()` (positions change only when laps are recorded), computed client-side.

### D2: Weather chip in the Race Control card header
A compact chip (icon + condition name) placed in the Race Control card header next to the status indicator. Populated from the latest entry of `GET /api/weather?race_id=0` on load and updated synchronously in `setWeather()`. Reuses the existing `weatherNames` map. Conditions card behavior is unchanged.

### D3: Next race countdown
Fetched once in `loadControllerData()` (same `race-info` call pattern) and re-rendered after `saveRaceSettings()`. Format: `Next race: 2026-08-22 · in 7 days` (or `· today` when the date is today, `· tomorrow` for +1). Hidden when `next_race_date` is empty or unparseable. Placed in the Configuration card's Race Settings section.

### D4: Tests
Playwright e2e: set weather → chip updates without opening Conditions; record lap positions → gaps appear with the leader showing "LEAD"; `next_race_date` set via API → countdown renders; empty states (no lap records, no next date) hide the respective elements. No Go tests (no backend change).

## Risks / Trade-offs

- [Gap column is empty early in a race] → Intended: laps are only known once recorded; hidden until then.
- [Gap semantics differ from time-based TV gaps] → TV's gap display is currently random placeholder data; this change does not touch TV. The lap-based gap is the honest representation available from stored data.
- [Chip duplicates Conditions card weather display] → Both update from the same source; the chip is the always-visible summary, the card remains the editor.
