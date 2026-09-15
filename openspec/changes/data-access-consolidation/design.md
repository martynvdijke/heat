## Context

Heat is a Go + Gin + Ent + raw SQLite app (SQLite with WAL — `heat.db-wal` present). `app/app.go` exposes both `DB *sql.DB` and `Ent *ent.Client` on `Server`. Raw `*sql.DB` is referenced in 45 non-test files; Ent in only 7. Generated Ent code lives under `ent/` (292 tracked files) with schemas in `ent/schema/`. Data-ish code is spread across `db/` (10 files: db.go, init.go, helpers.go, seed*.go, backup.go, race_history_season.go), `racing/` (11 files: stats.go, elo.go, consistency.go, head_to_head.go, qualifying.go, streaks.go, track_stats.go, cache.go, scope.go, csv_export.go, racing.go), `handlers/` (34 files), and `models/models.go` (490 lines). Migrations/init are in `db/init.go`, seeding in `db/seed*.go`, and OpenTelemetry instrumentation exists (`telemetry.go`, `middleware/tracing.go`). Ent was adopted but the codebase remains half-migrated with no single data-access convention.

## Goals / Non-Goals

**Goals:**

- Establish a single data-access convention: Ent as system of record for application reads/writes; raw `database/sql` confined to migrations/seeds/pragmas.
- Introduce per-domain repository boundaries so `handlers/` stops issuing raw SQL and depends on domain repos returning domain models.
- Define one transaction pattern covering both `database/sql` and `ent.Tx` semantics.
- Keep OpenTelemetry instrumentation consistent by placing it at the repository boundary.
- Migrate incrementally by vertical slice with safe rollback, without a big-bang rewrite.

**Non-Goals:**

- Big-bang port of all 45 raw-SQL call sites in this change.
- Removing `DB *sql.DB` from `app.Server` if a follow-up is safer (deferred).
- Changing SQLite/WAL or introducing a new database engine.
- Rewriting pure-stats logic in `racing/` beyond giving it data via interfaces.

## Decisions

### D1: Ent as system of record for application reads/writes; raw SQL only for migrations/seeds/pragmas

Rationale: Ent schemas in `ent/schema/` already define the domain model; Ent gives typed queries, code generation, and a single place to evolve the model. Raw `database/sql` remains necessary for DDL, pragmas, and bulk seed operations where Ent's abstraction adds no value. Choosing one system of record eliminates the double-cost of every DB change.

Alternatives considered: (a) Keep raw SQL as system of record and generate Ent only as a thin wrapper — rejected because Ent already owns 292 generated files and raw SQL has no single source of truth for model evolution. (b) Keep both indefinitely with ad-hoc choice per feature — rejected because it preserves the current half-migrated state and duplicate cost.

### D2: Repository / data-access functions per domain package vs methods on Server

Rationale: Prefer per-domain repositories (e.g. `racers.Repository`, `rounds.Repository`) that return domain models from `models/models.go` and are constructed with an Ent client. Handlers depend on an interface, not on `app.Server.DB` or `app.Server.Ent`. This keeps `handlers/` free of SQL, makes `racing/` pure-stats logic testable via interfaces, and avoids bloating `Server` with data methods. Domain packages (`db/` retains migration/seed, new or existing `racing/` subpackage for racing data access) own their repositories.

Alternatives considered: (a) Methods on `Server` (e.g. `func (s *Server) GetRacer(...)`) — rejected because `Server` already carries `DB`, `Ent`, `SessionStore`, websocket maps, and broadcast channels; adding data methods would further couple transport and persistence. (b) Single global `Store` struct — rejected because it recreates the same coupling with a different name and hides domain boundaries.

### D3: Incremental vertical-slice migration, no big-bang

Rationale: 45 raw-SQL call sites cannot be safely ported atomically without high regression risk, especially with SQLite WAL single-writer constraints. A vertical slice (e.g. racers or rounds) is inventoried, an interface is defined, the Ent-backed implementation is added behind it, handlers are switched, and the slice is tested before moving to the next. Both paths coexist behind the same interface until verified.

Alternatives considered: (a) Big-bang rewrite of all handlers to Ent — rejected due to large blast radius, difficult review, and no safe rollback. (b) No migration (document convention only) — rejected because it leaves the half-migrated state in place.

### D4: Single transaction pattern bridging database/sql and ent.Tx

Rationale: `database/sql` uses `sql.Tx` with explicit `Begin/Commit/Rollback`; Ent uses `ent.Tx` with its own lifecycle. Callers should not need to know which stack a repository uses. The pattern is: repositories expose transactional operations via a `Tx` abstraction or a `WithTx(ctx, func(txRepo) error)` helper; implementation maps `sql.Tx` and `ent.Tx` to the same interface; commit/rollback semantics are unified and a failed `Commit` always triggers rollback.

Alternatives considered: (a) Expose both `sql.Tx` and `ent.Tx` to callers — rejected because it leaks the dual-stack problem to every handler. (b) No cross-stack transactions (each repo manages its own) — rejected because multi-repo writes would lack atomicity.

### D5: Keep *sql.DB for migrations/seed/healthcheck only

Rationale: `db/init.go` (migrations) and `db/seed*.go` (seeding) plus SQLite pragmas and health checks genuinely need `database/sql`. Confining `*sql.DB` to `db/` prevents new raw SQL from leaking into handlers or racing stats. Application reads/writes go through Ent repositories; `db/` re-exports only `Migrate`, `Seed`, `HealthCheck`, and pragma helpers.

Alternatives considered: (a) Remove `*sql.DB` entirely and do migrations via Ent Atlas — rejected because bulk seed, backup (`db/backup.go`), and pragma logic are simpler and lower-risk with raw SQL today. (b) Keep `*sql.DB` generally available — rejected because it preserves the current unbounded raw-SQL surface.

### D6: Instrumentation at the repository boundary

Rationale: `telemetry.go` and `middleware/tracing.go` already instrument requests. Duplicating spans/metrics per handler is noisy and inconsistent. Placing tracing and metrics at the repository boundary (one span per repository call, with domain operation name) gives uniform observability regardless of caller and avoids double-instrumentation when a handler calls multiple repos.

Alternatives considered: (a) Keep instrumentation only in handlers/middleware — rejected because data-access latency and errors become invisible. (b) Instrument both handlers and repositories — rejected because it duplicates spans and doubles metric cardinality.

## Risks / Trade-offs

- SQL semantics drift when porting raw queries to Ent → Mitigation: inventory raw SQL usages first; port with side-by-side query comparison; keep raw path behind the same interface until verified; add focused regression tests per slice.
- Transaction/isolation differences between database/sql and ent.Tx → Mitigation: define one `WithTx` / `Tx` pattern (D4) with unified commit/rollback semantics; test rollback on failure; document SQLite transaction behavior.
- SQLite WAL single-writer concurrency limits → Mitigation: keep transactions short; avoid long-held `Tx` across handler boundaries; note WAL single-writer ceiling and sequence writes where needed.
- N+1 from Ent eager-loading (with/with vs explicit edges) → Mitigation: define loading policy per repository (explicit `With*` / predicate builders); add benchmarks or query-count assertions for hot paths; review Ent query plans per slice.
- Behavior regression without a safe test net → Mitigation: treat test-suite-restructure as a dependency; add repository-level tests and handler integration tests per vertical slice before cutting over; keep raw implementation behind interface for rollback.
- Half-migrated state persists longer than planned → Mitigation: vertical-slice plan with clear inventory and per-slice exit criteria; lint rule or grep in CI to prevent new raw SQL outside `db/`.

## Migration Plan

1. Inventory all raw SQL usages (the 45 non-test `.DB` references) and categorize by domain (racers, rounds, race history, etc.); publish the inventory as a checklist.
2. Define repository interfaces for the first vertical slice (e.g. racers or rounds) returning domain models from `models/models.go`; place them in the owning domain package.
3. Implement the Ent-backed repository for that slice; keep the existing raw-SQL path behind the same interface until verified.
4. Switch handlers for that slice to depend on the repository interface instead of `Server.DB`; wire via constructor or `Server` field that holds the interface.
5. Add repository-level tests and handler integration tests for the slice; verify tracing spans appear at the repository boundary.
6. Verify parity (query results, error handling, transaction rollback) and remove the raw-SQL implementation for the slice only after verification.
7. Repeat steps 2–6 for the next vertical slice until all application reads/writes go through Ent repositories.
8. Rollback: if a slice regresses, flip the interface binding back to the raw-SQL implementation without changing handlers; raw path remains available until the slice is deleted. Full rollback is re-binding all slices to raw implementations.
9. Follow-up: when all slices are migrated, remove `DB *sql.DB` from `app.Server` (or keep it confined to `db/` for migrations/seed/healthcheck if deferred per Open Questions).

## Open Questions

- Which vertical slice should go first — racers, rounds, or race-history/season — and what criteria (blast radius, test coverage, Ent schema readiness) decides the order?
- Should `racing/` pure-stats logic (elo.go, consistency.go, head_to_head.go, qualifying.go, streaks.go, track_stats.go, etc.) take data via repository interfaces vs direct DB, or remain pure functions that accept in-memory domain models?
- Whether to drop `DB` from `Server` struct in this change or a follow-up after all slices are verified and only `db/` needs it.
