## Context

Existing building blocks:

- `RaceEvent` (`handlers/race_enhancements.go`): `race_id, lap, event_type (overtake|crash|spin|safety_car|pit_stop), racer_id, racer_id2 (optional — overtake target), note, timestamp`; `POST /api/race-events` inserts and the controller polls `GET /api/race-events`.
- `SetWeather` POSTs a `models.WeatherCondition` and pushes it to `WeatherBroadcast`.
- `race_radio` is the closest existing pattern: table + `POST`/`GET` + `RaceRadioBroadcast` → `BroadcastRaceRadio` wraps the payload with `type:'race_radio'` before broadcasting.
- The `quotes` table powers the TV page's quote-of-the-day (`/api/quote/random`) — it is NOT the template source; templates are a separate, code-level concern (the audit doc's "Commentator Quotes database" does not exist as a template store).
- `ws.go` inbound switch handles `flag|self_service|lap_update|weather_update`; broadcasts are per-`Broadcast*` goroutine fed by server channels.

## Goals / Non-Goals

**Goals:**
- Store commentary entries (manual + auto-generated) with race chronology.
- Auto-narrate race events and weather changes; manual announcements from the race director.
- Real-time ticker on TV, spectator, and controller; polling fallback.
- Low ceremony: reuse the race_radio plumbing pattern end-to-end.

**Non-Goals:**
- Template management UI or DB-stored templates (hardcoded Go map for now).
- i18n of commentary templates (English only in this change).
- Auto-generation from flags (yellow/red/chequered) — manual only, to avoid noise.
- Audio/voice narration.

## Decisions

### D1: `commentary` table + startup migration
`commentary(id INTEGER PRIMARY KEY AUTOINCREMENT, race_id INTEGER NOT NULL DEFAULT 0, lap INTEGER NOT NULL DEFAULT 0, racer_id INTEGER NULL, message TEXT NOT NULL, template_key TEXT NULL, created_at TEXT NOT NULL DEFAULT (datetime('now')))`. Migration appended to the existing startup migration sequence in `db/init.go` (same style as the `next_race_date` column migration).

### D2: Auto-generation hooks
- `AddRaceEvent`: after a successful insert, generate + insert + broadcast a commentary entry for the event type.
- `SetWeather`: after upsert, generate + insert + broadcast a condition-change entry ("Conditions change: Wet" style, referencing lap).
Flags are deliberately excluded (transient, high-frequency).

### D3: Template engine (Go, hardcoded)
`map[eventType][]string` with 2–3 variants each, placeholders `{{driver}}`, `{{target}}` (from `racer_id2` when present, else generic), `{{lap}}`. Random variant per generation. Weather: single template family keyed by condition. Manual entries are stored verbatim (`template_key` NULL). No DB table for templates; future i18n is a non-goal.

### D4: WS broadcast wrapper
New `CommentaryBroadcast` channel on the server; `BroadcastCommentary` goroutine wraps each entry as `{type:'commentary', ...entry}` exactly like `BroadcastRaceRadio`. Inbound WS does not need a new message type (commentary is never pushed *from* clients over WS).

### D5: Ticker behavior
`CommentaryTicker` in `ts/commentary.ts`: appends entries to a scroll container; WS message → append immediately; polling every 5s with `since=<lastId>` catches up after reconnect or for pages without WS; entries older than 30s fade out (CSS transition, then removed); pause on hover; `aria-live="polite"` on the container.

### D6: Placement
- TV: ticker strip below the leaderboard.
- Spectator: section above the events area.
- Controller: compact feed + "Commentary" manual input (message + optional driver select) in the Tracking card; manual POST to `/api/commentary`.

### D7: Tests
Go: POST manual entry → stored + GET returns it; GET filters (`race_id`, `since`, `limit`); POST race event → auto-generated entry exists with substituted driver name; POST weather → entry exists. Playwright: controller manual entry → appears on TV ticker (via WS); old entries fade (waited-out).

## Risks / Trade-offs

- [Event/weather spam flooding the ticker] → One entry per event/weather change; fade-out keeps the UI bounded; `limit` caps the poll response.
- [Pages without WS lag] → 5s poll with `since` catch-up; TV/spectator both have WS today (controller gains WS in the controller-start-lights change).
- [Template quality] → Variants per event type keep messages varied; templates stay in code, so tuning is a code change (accepted for v1).
