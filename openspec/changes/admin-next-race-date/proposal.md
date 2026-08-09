## Why

Admins currently cannot communicate the date of the next race event to racers — the app tracks live race info (country, track, laps) but has no notion of an upcoming race date. There is no persistent, settable value for "when is the next race event."

## What Changes

- Add a persistent **next race date** value that admins can set from the admin panel.
- Add a date field ("Next Race Date") to the Race Day tab's "Race Information" form in the admin UI.
- Extend the existing `/api/race-info` GET/POST endpoints to read and write the next race date alongside the current race info.
- Store the value in the existing `race_info` table via an idempotent `ALTER TABLE` migration (matching the project's migration pattern in `db/init.go`).
- Seed the field as empty (nullable) so existing installs and fresh databases work without changes.

No public-facing display, TRMNL payload, or email behavior changes — this change is scoped to making the date settable and retrievable.

## Capabilities

### New Capabilities
- `next-race-date`: Admins can set and read the date of the next race event from the admin panel; the value persists in the database and is exposed through the race info API.

### Modified Capabilities
<!-- None: no existing spec changes behavior at the requirement level. -->

## Impact

- `db/init.go` — add idempotent `ALTER TABLE race_info ADD COLUMN next_race_date` migration.
- `models/models.go` — add `NextRaceDate` field to `models.RaceInfo`.
- `handlers/race.go` — update `GetRaceInfo` / `UpdateRaceInfo` to read/write the new column.
- `static/templates/tab-race-day.html` — add a date input to the Race Information form.
- `ts/admin.ts` (compiled to `static/js/admin.js`) — load/save the new field in the race form handlers.
- Tests: `04_test_race_test.go` covers race-info endpoints; add coverage for the new field round-trip.
