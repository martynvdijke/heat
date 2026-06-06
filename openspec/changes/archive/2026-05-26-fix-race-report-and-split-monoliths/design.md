## Context

The HEAT codebase grew organically. Four key files have become monoliths handling multiple concerns:

| File | Lines | Concerns |
|---|---|---|
| `racing/racing.go` | 805 | 14 unrelated racing computation functions |
| `handlers/stats.go` | 519 | 14 HTTP handler functions |
| `middleware/middleware.go` | 158 | 6 middleware types + a custom response writer |
| `db/db.go` | 289 | Init, backups, 10 seed functions, utility helpers |

Additionally, `racing.RaceReport()` (lines 690-756) has a broken frontend/backend data contract, and `racing.Streaks()` (lines 354-373) runs a duplicate SQL query.

No architectural changes — all refactoring is internal. No new dependencies.

## Goals / Non-Goals

**Goals:**
- Fix the race report frontend/backend field mismatches
- Fix the duplicate query bug in `Streaks()`
- Break the 4 monolith files into focused, single-concern files
- Remove dead code paths (unused struct fields, duplicate functions, stdout telemetry noise)

**Non-Goals:**
- Changing the Ent ORM schema or models
- Changing any API contracts or frontend behavior (other than fixing bugs)
- Adding new features or capabilities
- Performance optimization (beyond eliminating the duplicate query)
- Adding tests for untested paths

## Decisions

### Race Report Fix Approach

**Keep the page and API, fix the data contract, remove dead code.**

The race report API (`GET /api/race-report`) returns a `RaceReportData` struct. The frontend JS expects fields that don't match. Fix:

1. **Backend**: Add `country` and `total_laps` to the SQL query from `race_history` table. Change JSON tag `race_name` → `name` (or add a `name` computed field). The `RaceHistory` model already has `Country` and `TotalLaps` fields.
2. **Struct**: Remove the never-populated `LapRecords` and `RaceRadio` fields.
3. **Frontend**: Fix JS to use correct keys (`race_name`, `dns` instead of `did_not_start`). Remove the `window.print()` auto-trigger.
4. **RaceResultEntry**: Keep the struct for now (only used by report). If later unused, remove entirely.

**Alternative considered**: Deleting the feature entirely. Rejected per user preference — keep and fix.

### Streaks() Query Fix

**Merge two queries into one.**

The first query already selects all 4 needed fields (`wins`, `podiums`, `total`, `avg`). Fix the `Scan()` call to capture `totalRaces` and `avgPosition` into proper variables instead of `new(int)` / `new(float64)`. Delete the redundant second query.

### Monolith Splitting Strategy

**One file per concern within the same package.** No new packages, no import changes needed (everything is internal to the package).

- `racing/`: Each domain gets its own file. The `RaceReport` function stays in its own file (or the race report fix file) since it's being fixed.
- `handlers/`: Split `stats.go` into domain-grouped files.
- `middleware/`: Each middleware function type gets its own file, plus the shared `umamiResponseWriter` stays with the Umami middleware.
- `db/`: Init + backup + seeds split into focused files.

### Duplicate Function Consolidation

`RacerStatsFallback()` and `AllRacerStats()` both query `racer_stats` with slightly different column selection. These are legacy wrappers. Remove `RacerStatsFallback()` and rename `AllRacerStats()` to be the canonical function. Same for `SingleRacerStatsFallback()` — it's unclear if anything still calls it (check callers first).

### OpenTelemetry

Remove the stdout trace exporter — it generates noisy JSON output that nobody reads. Keep the Prometheus metrics (they're useful). If tracing is needed later, a proper exporter (Jaeger, Zipkin) should be configured instead.

## Risks / Trade-offs

- **Risk: Split files break git blame** → Acceptable trade-off. The improved navigation outweighs history loss.
- **Risk: Import path changes** → None — all splits stay within the same Go package. No import updates needed.
- **Risk: Race report fix breaks if API consumers depend on old field names** → Extremely low — the API has no external consumers. The only consumer is the bundled `race-report.html` page, which we're fixing simultaneously.
- **Risk: `RacerStatsFallback` is called from tests or handlers** → Check callers before removing. If called, inline or redirect to the canonical function.
