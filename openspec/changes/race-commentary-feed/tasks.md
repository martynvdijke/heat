## 1. Backend: storage + API

- [x] 1.1 Add `commentary` table migration to `db/init.go` (startup migration style)
- [x] 1.2 Create `handlers/commentary.go`: `POST /api/commentary` (manual entry), `GET /api/commentary` (`race_id`, `since`, `limit`, newest-first)
- [x] 1.3 Register routes in `main.go` and add the `CommentaryBroadcast` channel to the server

## 2. Backend: auto-generation + WS

- [x] 2.1 Template engine (Go map per event_type, 2–3 variants, `{{driver}}`/`{{target}}`/`{{lap}}` substitution; weather templates; random variant)
- [x] 2.2 Hook generation into `AddRaceEvent` and `SetWeather` (insert + broadcast)
- [x] 2.3 `ws/ws.go`: `BroadcastCommentary` goroutine wrapping entries as `{type:'commentary', ...}` (mirror `BroadcastRaceRadio`); start it in `main.go`

## 3. Frontend: ticker

- [x] 3.1 Create `ts/commentary.ts`: `CommentaryTicker` (WS append, 5s poll with `since`, 30s fade-out, hover pause, `aria-live="polite"`)
- [x] 3.2 `static/tv.html` + `ts/tv.ts`: ticker below the leaderboard
- [x] 3.3 `static/spectator.html` + `ts/spectator.ts`: commentary section above events
- [x] 3.4 `static/controller.html` + `ts/controller.ts`: compact feed + manual entry input in the Tracking card

## 4. Tests

- [x] 4.1 Go tests: manual POST/GET round-trip; `since`/`limit`/`race_id` filters; auto-generation from race event (driver substitution) and weather change
- [x] 4.2 Playwright e2e: manual entry from controller appears on TV ticker; old entries fade
- [ ] 4.3 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
