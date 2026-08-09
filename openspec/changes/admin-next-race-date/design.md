## Context

The "heat" app (Go + Gin + SQLite) manages racing events. Admins configure the current race on the **Race Day** admin tab, which loads and saves race info through `GET/POST /api/race-info`. The data lives in the `race_info` table (`country`, `track`, `track_id`, `laps`). The app has no notion of an upcoming race date.

The codebase has an established pattern for persistent single-value settings: single-row tables seeded in `db/seed_settings.go` (e.g., `umami_settings`, `notification_settings`), model structs in `models/models.go`, GET/POST handlers in `handlers/settings.go`, and routes under the admin group in `main.go`. Schema additions use idempotent `ALTER TABLE` statements in `db/init.go` (errors are ignored, matching the existing pattern at lines 54–65).

## Goals / Non-Goals

**Goals:**
- Store a next race date persistently so it survives restarts.
- Let admins set and clear the date from the admin panel.
- Expose the date through the existing race info API so the UI can read and write it.

**Non-Goals:**
- Displaying the date on public/TV/pitboard pages, TRMNL e-ink payload, or emails/notifications.
- Countdown or reminder behavior.
- Validating that the date is in the future (admins may backdate for planning).

## Decisions

**1. Store the date in the existing `race_info` table (not a new settings table).**
The next race date is a property of the current race event, which is already managed on the Race Day tab. Adding a nullable `next_race_date TEXT` column keeps the value with the data it describes and requires no new seed function, model, or dedicated endpoints.
*Alternative considered:* a new single-row `race_settings` table following the `umami_settings` pattern. Rejected — it adds a table, seed, model, and two endpoints for a single value whose natural home is the race info the admin already edits.

**2. Extend `GET/POST /api/race-info` instead of adding dedicated endpoints.**
`models.RaceInfo` gains `NextRaceDate string \`json:"next_race_date"\``. The GET handler reads the column (`COALESCE(next_race_date, '')`), the POST handler writes it. The Race Day tab already loads this endpoint on activation and submits the form to it, so frontend wiring is a single field.
*Alternative considered:* dedicated `GetNextRaceDate`/`SaveNextRaceDate` settings endpoints. Rejected — more routes and client calls for no benefit; the value is conceptually race info.

**3. Migration via idempotent `ALTER TABLE` in `db/init.go`.**
`ALTER TABLE race_info ADD COLUMN next_race_date TEXT NOT NULL DEFAULT ''` placed alongside the existing ALTER statements. Errors are ignored (existing pattern tolerates "duplicate column" on re-runs). The default `''` (empty string) means "not set" — consistent with how other TEXT columns default.

**4. Format and validation: `YYYY-MM-DD` text, empty means unset.**
The admin UI uses `<input type="date">`, which enforces the format. The server validates with `time.Parse("2006-01-02", ...)` and rejects invalid values with `400`, so the API can't be polluted by arbitrary strings.

## Risks / Trade-offs

- **Duplicate-column error on upgrade re-runs** → `db/init.go` already runs these statements without error checks; the existing pattern tolerates the failure, so re-running is safe.
- **Stale frontend bundle** → `static/js/admin.js` is compiled from `ts/admin.ts`; changes must be built (Taskfile `build`/`pre-push` compiles TS) before the field appears in the admin UI.
- **TEXT column is format-loose at the DB level** → mitigated by server-side parsing validation (Decision 4) and the `type="date"` input.
- **Value drifts when the next race happens** → out of scope; admins update the field each race cycle, same as they already update track/laps.

## Migration Plan

1. Deploy code (migration runs at startup; existing rows default to `''` = unset).
2. No data backfill needed — nullable-by-default means no existing behavior changes.
3. Rollback: remove the column read/write from handlers and the field from the UI; leaving the column in the DB is harmless (unused TEXT column).
