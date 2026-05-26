## Why

The codebase has accumulated technical debt during organic growth — a race report page with a broken frontend/backend data contract, a duplicate SQL query bug in streak calculations, four monolith files handling multiple concerns, and dead code paths. These make the code harder to maintain and navigate without providing value. This change fixes the broken feature, resolves the bug, and reorganizes the codebase for clarity.

## What Changes

- **Fix the race report**: Align the frontend JSON field expectations with the Go struct, add missing fields (country, total_laps) to the API query, remove never-populated struct fields (LapRecords, RaceRadio), remove intrusive auto-print behavior
- **Fix duplicate SQL query in `Streaks()`**: Merge two near-identical queries into one, fix the `Scan()` call that discards results
- **Split `racing/racing.go`**: Break the 805-line monolith into separate files by domain (stats, track stats, streaks, ELO, qualifying, consistency, head-to-head, CSV export)
- **Split `handlers/stats.go`**: Break the 519-line, 14-handler file into domain-grouped files
- **Split `middleware/middleware.go`**: Separate the 4 middleware concerns (security, CSRF, auth, rate limiting, Umami) into individual files
- **Split `db/db.go`**: Separate initialization, backup, seed functions, and helpers into their own files
- **Remove duplicate functions**: Consolidate `RacerStatsFallback`/`AllRacerStats` and `SingleRacerStatsFallback`/`SingleRacerStatsBySeason` where appropriate
- **Strip noisy stdout OpenTelemetry tracing**: Stop exporting trace data to stdout (nobody reads it)
- **Replace custom `EscapeHTML()`**: Use `html.EscapeString()` from stdlib instead

## Capabilities

No new capabilities. This is purely a code health and bug-fix change — all changes are internal implementation details with no API contract modifications.

### Modified Capabilities

None. The race report API contract is being fixed but not fundamentally changed. No spec-level behavior changes.

## Impact

- `racing/racing.go`: Restructured into multiple files; `RaceReport()` function fixed; `Streaks()` query fixed; duplicate functions consolidated
- `handlers/stats.go`: `GetRaceReport()` handler fixed; file restructured into domain-grouped files
- `middleware/middleware.go`: Restructured into individual files per middleware function
- `db/db.go`: Restructured into focused files
- `static/race-report.html`: Modified to fix data display and remove auto-print
- `static/index.html`: No change (nav link stays)
- `static/locales/en.json`, `static/locales/nl.json`: No change (translations stay)
- Tests: Updated to match new structures
