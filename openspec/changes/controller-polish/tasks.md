## 1. Weather chip

- [x] 1.1 Add weather chip markup to the Race Control card header in `static/controller.html`
- [x] 1.2 `ts/controller.ts`: fetch `GET /api/weather?race_id=0` in `loadControllerData()`, render latest entry into the chip
- [x] 1.3 Update the chip in `setWeather()` alongside the existing `#current-weather` text

## 2. Gap to leader

- [x] 2.1 `ts/controller.ts`: fetch `GET /api/lap-records?race_id=0` on load and after `recordCurrentLap()`; compute per-driver completed laps and leader laps
- [x] 2.2 `renderStandings()`: render "LEAD" / "+N" / empty gap per the rules; skip gap column when no lap data
- [x] 2.3 Minor CSS for the gap cell in `static/style.css`

## 3. Next race countdown

- [x] 3.1 Add next-race line markup to the Configuration card in `static/controller.html`
- [x] 3.2 `ts/controller.ts`: read `next_race_date` from `GET /api/race-info` in `loadControllerData()`; render countdown; re-render after `saveRaceSettings()`; hide when unset/unparseable

## 4. Tests

- [x] 4.1 Playwright e2e: chip updates on set-weather; gaps appear after recording laps (leader "LEAD", +N for lapped drivers); countdown renders from a set `next_race_date` and hides when unset
- [x] 4.2 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
