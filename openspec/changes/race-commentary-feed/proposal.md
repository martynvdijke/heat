## Why

Race events (overtakes, crashes, spins, safety car, pit stops), flags, and weather changes are recorded and broadcast, but never narrated — the TV and spectator views show raw data only. `docs/design-audit-proposals.md` (2026-05-24) documents a "Race Director Commentary Feed": an automatic narrative ticker generated from race events with a quote template engine, manual entries from the race director, and timestamped chronology. It was never built. The `race_radio` pattern (table + WS broadcast with a `type` wrapper) provides a proven template for the same plumbing.

## What Changes

- **New `commentary` table** (`id`, `race_id`, `lap`, `racer_id` NULL, `message`, `template_key` NULL, `created_at`) with a migration in `db/init.go` following the existing startup-migration pattern.
- **New `handlers/commentary.go`**:
  - `POST /api/commentary` — manual entry (message, optional racer_id/lap) → insert + WS broadcast.
  - `GET /api/commentary?race_id=&since=&limit=` — newest-first list; `since` = last seen entry id for polling/reconnect catch-up.
- **Auto-generation**: adding a race event (`AddRaceEvent`) or changing weather (`SetWeather`) generates a commentary entry via a Go template engine (per-event-type template variants with `{{driver}}`, `{{target}}`, `{{lap}}` substitution, randomized variant) and broadcasts it.
- **WS broadcast** `type:'commentary'`: new `CommentaryBroadcast` channel + `BroadcastCommentary` goroutine in `ws/ws.go` (mirrors `BroadcastRaceRadio`, including the `type` wrapper).
- **New `ts/commentary.ts`**: `CommentaryTicker` — renders entries, consumes WS `type:'commentary'`, polls `GET /api/commentary?since=<lastId>` every 5s as fallback/reconnect, fades entries older than 30s, pauses on hover.
- **UI**: ticker on `tv.html` (below the leaderboard), section on `spectator.html` (above events), and on the controller a compact feed + manual entry input in the Tracking card.
- **Tests**: Go handler tests (manual POST, GET filters, auto-generation on events/weather) and a Playwright e2e (manual entry from the controller appears on the TV ticker).

## Capabilities

### New Capabilities
- `race-commentary`: Commentary entry storage and API, automatic generation from race events and weather changes, WS broadcast, and the ticker UI on TV/spectator/controller.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `db/init.go`: `commentary` table migration.
- `handlers/commentary.go` (new); `handlers/race_enhancements.go`: generation hooks in `AddRaceEvent` + `SetWeather`; template engine (new `handlers/commentary_templates.go` or inline).
- `ws/ws.go`: `CommentaryBroadcast` channel + `BroadcastCommentary` goroutine; `app` Server fields in `main.go` + route registration.
- `ts/commentary.ts` (new); `ts/tv.ts`, `ts/spectator.ts`, `ts/controller.ts` integration; `static/tv.html`, `static/spectator.html`, `static/controller.html` markup.
- Tests: Go + Playwright.
