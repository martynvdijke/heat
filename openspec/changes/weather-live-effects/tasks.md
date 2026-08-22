## 1. Backend

- [x] 1.1 `handlers/stats_performance.go`: pace-heatmap query LEFT JOINs `weather_conditions` (race_id + `lap_number BETWEEN lap_start AND lap_end`), adds `condition` + `grip_modifier` to the response (COALESCE dry/1.0)
- [x] 1.2 `ws/ws.go`: `BroadcastWeather` wraps broadcasts as `{type:'weather_update', ...}` (BroadcastRaceRadio pattern)
- [x] 1.3 Go tests: pace-heatmap annotation (wet range vs outside, empty weather); `SetWeather` persists `lap_end`

## 2. TV + pitboard banners

- [x] 2.1 `static/tv.html` + `ts/tv.ts`: color-coded weather banner (icon, name, grip %), WS `weather_update` handler + initial fetch, forecast line
- [x] 2.2 `static/pitboard.html` + `ts/pitboard.ts`: banner upgrade (condition, grip %, forecast), refresh via existing fetch mechanism

## 3. Spectator + controller

- [x] 3.1 `static/spectator.html` + `ts/spectator.ts`: forecast line next to the existing condition text
- [x] 3.2 `static/controller.html` + `ts/controller.ts`: "Until Lap" (`lap_end`) input in Conditions card; `setWeather()` sends it; active + upcoming entries listed after save

## 4. Stats page heatmap

- [x] 4.1 `ts/stats.ts` + `static/templates/stats.html`: per-cell weather badge, legend, hover tooltip with condition + grip %

## 5. Tests

- [x] 5.1 Playwright e2e: set wet weather → TV banner shows Wet + grip %; scheduled future entry → forecast line on TV; controller saves `lap_end` and lists active/upcoming
- [x] 5.2 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
