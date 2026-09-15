## Why

`app/app.go` (130 lines) defines `type Server struct` exposing BOTH `DB *sql.DB` and `Ent *ent.Client` (plus `SessionStore`, websocket client maps, and broadcast channels). Raw `*sql.DB` (`.DB`) is referenced in 45 non-test files while Ent (`.Ent`) is referenced in only 7. Generated Ent code lives under `ent/` (292 tracked files) with schemas in `ent/schema/`. The codebase is half-migrated: two data-access vocabularies coexist, every DB change costs twice, and Ent adoption has stalled. Without a single convention, handlers continue to issue raw SQL, migrations and business logic share the same `*sql.DB` handle, and new contributors have no clear system of record.

## What Changes

- Establish one convention: Ent is the system of record for application reads/writes; raw `database/sql` is confined to a thin `db/` layer for migrations, seeding, and SQLite pragmas.
- Introduce per-domain data-access boundaries (repository / data-access functions per domain package) so `handlers/` stops issuing raw SQL and instead calls domain repos returning domain models.
- Migrate incrementally by vertical slice (e.g. racers, then rounds, etc.) — no big-bang rewrite; both access paths coexist behind the same repository interface until a slice is verified.
- Define a single transaction pattern that bridges `database/sql` and `ent.Tx` so callers do not need to know which stack they are in.
- Keep `*sql.DB` for migrations/seed/healthcheck only; eventually stop exposing both `DB` and `Ent` on `app.Server` (deferred removal if not safe in this change).
- Instrument at the repository boundary so OpenTelemetry tracing/metrics (`telemetry.go`, `middleware/tracing.go`) are not duplicated per handler.

## Capabilities

### New Capabilities

- `data-access-layer`: Single Ent-backed data-access convention with per-domain repository boundaries, unified transaction handling, and repository-level observability; raw SQL confined to migration/seed/pragma layer.

### Modified Capabilities

None.

## Impact

Backend data-access across `db/` (10 files: db.go, init.go, helpers.go, seed*.go, backup.go, race_history_season.go), `racing/` (11 files: stats.go, elo.go, consistency.go, head_to_head.go, qualifying.go, streaks.go, track_stats.go, cache.go, scope.go, csv_export.go, racing.go), `handlers/` (34 files), `models/models.go` (490 lines), and `app/app.go`. No frontend or schema-breaking change in the first slice; follow-up slices port remaining raw SQL usages to Ent. Risk of behavior regression mitigated by interface-preserving migration and dependency on test-suite restructure.
