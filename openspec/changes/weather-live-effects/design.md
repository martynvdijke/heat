## Context

- `weather_conditions(race_id, condition, lap_start, lap_end, grip_modifier)` — `SetWeather` (handlers/race_enhancements.go) INSERTs when `id==0`, UPDATEs otherwise, then pushes the model to `WeatherBroadcast`. Controller sends `lap_end` hardcoded to 999 and a gripMap (dry 1.0, damp 0.85, wet 0.7, torrential 0.5). `GetWeather` returns all entries for a `race_id` ordered by `lap_start`.
- `BroadcastWeather` (ws/ws.go) sends the **bare** `models.WeatherCondition` — no `type` field — so today no client can distinguish it from other non-array messages. `ts/tv.ts`'s `onmessage` checks `Array.isArray`, `type==='flag'`, `type==='self_service'`, `type==='sound'` — a bare weather object is silently ignored.
- `GET /api/stats/pace-heatmap` (handlers/stats_performance.go) selects `lap_records` fields (position/gear/heat/turbo) with an optional `racer_id` filter, ordered by racer + lap, limit 1000. `lap_records` has **no lap_time column** — pace is positional, not time-based.
- Spectator (`ts/spectator.ts`) renders `spec-weather` from `GetSpectatorState`; pitboard (`ts/pitboard.ts`) fetches `GET /api/weather?race_id=0` on load and shows icon + capitalized name.

## Goals / Non-Goals

**Goals:**
- Weather context in the pace heatmap data and UI.
- Prominent live weather banner on TV and pitboard, forecast previews from scheduled entries.
- Controller can schedule weather changes (`lap_end`) and see active + upcoming entries.
- Make the weather WS broadcast identifiable (`type:'weather_update'`).

**Non-Goals:**
- Grip-adjusted lap-time math — `lap_records` stores no lap times, so "wet laps are X% slower" computations are impossible with current data (noted explicitly so nobody assumes otherwise).
- Weather affecting gameplay/flag logic server-side.
- New tables or schema changes (columns and API shape already exist).
- Per-lap weather entry editing UI beyond the existing Conditions card.

## Decisions

### D1: Pace heatmap annotation (backend, additive)
Extend the pace-heatmap query with `LEFT JOIN weather_conditions wc ON wc.race_id = lr.race_id AND lr.lap_number BETWEEN wc.lap_start AND wc.lap_end`, selecting `COALESCE(wc.condition, 'dry')` and `COALESCE(wc.grip_modifier, 1.0)`. Response gains `condition` + `grip_modifier` per point; existing fields unchanged (backward compatible). Multiple overlapping entries (rare) resolve to the first match — acceptable.

### D2: Broadcast wrapper `type:'weather_update'`
`BroadcastWeather` broadcasts `{type:'weather_update', ...wc}` (map wrapper like `BroadcastRaceRadio`) so clients can route it. Inbound WS `weather_update` handling in `ws.go` is unchanged. TV/spectator/pitboard then handle the typed message; pitboard (polling today) additionally refreshes via its existing fetch on a short interval if no WS is present — TV uses WS + initial fetch.

### D3: Active/forecast rule
Current lap for display purposes: controller's `current-lap` / TV's `tv-lap`; when unknown, the latest entry by `lap_start` wins.
- **Active**: entry with `lap_start <= lap < lap_end` (999 = open-ended).
- **Forecast**: entries with `lap_start > lap`, earliest first → "Rain from lap N" (condition name in the label).
TV banner shows active condition (icon, name, grip % — e.g. "🌧️ Wet · 70% grip"), color-coded; forecast line shows the next scheduled change. Same rule client-side everywhere, computed from the `GET /api/weather?race_id=0` list.

### D4: Controller `lap_end` scheduling
Conditions card gains "Until Lap" number input (default 999, min = from-lap). `setWeather()` sends `lap_end`. After saving, the card lists active entry + upcoming entries ("Dry → Wet at lap 10"). Existing `#current-weather` display unchanged.

### D5: Stats heatmap UI
`ts/stats.ts` pace heatmap renders a small weather badge per cell (icon per condition) + a legend; hover tooltip includes condition + grip %. No layout change when all data is dry.

### D6: Tests
Go: pace-heatmap returns `condition`/`grip_modifier` for laps inside a wet entry and `dry`/`1.0` outside; `SetWeather` persists `lap_end`. Playwright: set wet weather → TV banner shows Wet + grip %; schedule future entry → forecast line appears; controller shows active/upcoming after save.

## Risks / Trade-offs

- [Bare weather broadcast change] → Wrapping adds a `type` field; no existing client parses the bare object (verified: TV ignores it today), so the change is safe.
- [Overlapping weather entries] → First-match JOIN; UI lists multiple entries so operators can clean up.
- [race_id mismatch between weather (0) and lap_records (0)] → Both default to 0 for the current race; JOIN matches. Historic races without weather rows fall back to dry/1.0.
