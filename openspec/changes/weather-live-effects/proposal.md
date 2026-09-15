## Why

Weather is stored with a `grip_modifier` (dry 1.0, damp 0.85, wet 0.7, torrential 0.5 via the controller's gripMap) and broadcast over WS, but it only affects two small UI spots: the spectator page shows a text line and the pitboard shows an icon + name. The TV page — the main viewing surface — shows **no weather at all**, the controller hardcodes `lap_end: 999` so future weather changes cannot be scheduled, and the pace heatmap data carries no weather context even though `weather_conditions` rows span explicit lap ranges. The grip data exists but is never surfaced.

## What Changes

- **Pace heatmap annotated with weather**: `GET /api/stats/pace-heatmap` LEFT JOINs `weather_conditions` (on `race_id` + lap range) so each pace point includes `condition` and `grip_modifier` (NULL → `dry`/`1.0`); the stats page pace heatmap renders a per-cell weather badge and a legend.
- **TV weather banner**: prominent color-coded banner (icon, condition name, grip %) below the TV header; driven by WS `weather_update` messages and an initial fetch. Requires wrapping the weather broadcast with `type:'weather_update'` (today `BroadcastWeather` sends the bare `WeatherCondition` JSON, which clients cannot distinguish from other messages).
- **Forecast display**: when a scheduled future weather entry exists (`lap_start` beyond the current lap), TV/spectator/pitboard show "Rain from lap N"-style preview lines from the same `GET /api/weather` list.
- **Pitboard banner**: upgrade the existing icon+text row to a banner including grip % and forecast (same data rule).
- **Controller weather scheduling**: the Conditions card gains a "Until Lap" (`lap_end`) input (default 999) so operators schedule changes ahead of time; after saving, active and upcoming entries are listed.
- **Tests**: Go tests for the annotated pace heatmap and `lap_end` persistence; Playwright e2e for the TV banner + forecast and the controller scheduling.

## Capabilities

### New Capabilities
- `weather-live-effects`: Weather visibility and scheduling — annotated pace heatmap, TV/pitboard banners, forecast previews, and controller `lap_end` scheduling.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `handlers/stats_performance.go`: pace-heatmap query + response shape (additive fields).
- `ws/ws.go`: `BroadcastWeather` wraps broadcasts with `type:'weather_update'`.
- `ts/tv.ts` + `static/tv.html`: banner + forecast + WS handling.
- `ts/pitboard.ts` + `static/pitboard.html`: banner + forecast.
- `ts/spectator.ts`: forecast line (existing condition text stays).
- `static/controller.html` + `ts/controller.ts`: `lap_end` input + active/upcoming list.
- `ts/stats.ts` + `static/templates/stats.html`: pace heatmap weather badges + legend.
- Tests: Go + Playwright.
