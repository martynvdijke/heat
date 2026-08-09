## 1. Database Migration

- [x] 1.1 Add idempotent `ALTER TABLE race_info ADD COLUMN next_race_date TEXT NOT NULL DEFAULT ''` to `db/init.go` alongside the existing ALTER TABLE statements

## 2. Backend Model and Handlers

- [x] 2.1 Add `NextRaceDate string \`json:"next_race_date"\`` field to `models.RaceInfo` in `models/models.go`
- [x] 2.2 Update `GetRaceInfo` in `handlers/race.go` to select `COALESCE(next_race_date, '')` and populate the new field
- [x] 2.3 Update `UpdateRaceInfo` in `handlers/race.go` to validate the submitted `next_race_date` with `time.Parse("2006-01-02", ...)` (reject invalid values with 400) and persist it in the INSERT

## 3. Admin UI

- [x] 3.1 Add a "Next Race Date" `<input type="date">` field to the Race Information form in `static/templates/tab-race-day.html`
- [x] 3.2 Update `ts/admin.ts` race info loader (`fe()`) to populate the field from `next_race_date`, and the race-form submit handler to include it in the POST body
- [x] 3.3 Rebuild the TypeScript bundle so `static/js/admin.js` reflects the changes

## 4. Tests

- [x] 4.1 Add coverage in `04_test_race_test.go`: `GET /api/race-info` returns `next_race_date` (empty when unset, stored value when set)
- [x] 4.2 Add coverage that `POST /api/race-info` with a valid date persists and round-trips, and with an empty date clears the value
- [x] 4.3 Add coverage that `POST /api/race-info` with an invalid date format returns 400 and does not change the stored value

## 5. Verification

- [x] 5.1 Run `task test` (Go tests) and TS compilation to confirm everything passes
- [x] 5.2 Run `task pre-push` to complete the full checklist (fmt, tests, vet + govulncheck, TS compile, build)
