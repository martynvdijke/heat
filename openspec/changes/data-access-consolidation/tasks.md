## 1. Inventory and scaffolding

- [ ] 1.1 Inventory all 45 non-test `Server.DB` / `*sql.DB` references and categorize by domain (racers, rounds, race history, etc.); publish checklist in the change.
- [ ] 1.2 Define repository interface shape for the first vertical slice (e.g. racers or rounds) returning domain models from `models/models.go`; place in owning domain package.
- [ ] 1.3 Add lint/grep guard (CI) to prevent new raw `database/sql` usage outside `db/` after the convention is established.

## 2. Transaction and observability foundation

- [ ] 2.1 Define single transaction pattern bridging `sql.Tx` and `ent.Tx` (`WithTx(ctx, func(txRepo) error)` or `Tx` abstraction) with unified commit/rollback semantics.
- [ ] 2.2 Implement repository-boundary OpenTelemetry instrumentation (span + metrics) reusing `telemetry.go` / `middleware/tracing.go`; verify one span per repository call.
- [ ] 2.3 Document SQLite WAL single-writer constraints and transaction-lifetime guidance for repository callers.

## 3. First vertical slice (e.g. racers or rounds)

- [ ] 3.1 Implement Ent-backed repository for the chosen slice behind the interface defined in 1.2; keep existing raw-SQL implementation behind the same interface.
- [ ] 3.2 Switch `handlers/` call sites for that slice to depend on the repository interface instead of `Server.DB`.
- [ ] 3.3 Add repository-level tests and handler integration tests for the slice; include query-result parity and N+1 / eager-loading checks.
- [ ] 3.4 Verify transaction commit/rollback for the slice and confirm tracing spans at the repository boundary.
- [ ] 3.5 Remove raw-SQL implementation for the slice only after parity is verified; keep interface binding for rollback.

## 4. Incremental migration of remaining slices

- [ ] 4.1 Repeat 3.1–3.5 for the next domain slice until all application reads/writes go through Ent repositories.
- [ ] 4.2 Confine `*sql.DB` usage to `db/` (migrations in `db/init.go`, seeding in `db/seed*.go`, pragmas/backup/healthcheck); remove or relocate any remaining raw SQL outside `db/`.
- [ ] 4.3 Decide whether `racing/` pure-stats modules (elo, consistency, head_to_head, qualifying, streaks, track_stats) consume data via repository interfaces vs in-memory domain models; implement the chosen approach.

## 5. Server struct and cleanup

- [ ] 5.1 Evaluate removing `DB *sql.DB` from `app.Server` (130 lines) vs keeping it confined to `db/` for migrations/seed/healthcheck; implement the chosen option or defer with a follow-up issue.
- [ ] 5.2 Run `openspec validate data-access-consolidation --strict` and fix until valid; ensure no `.openspec.yaml` was created and no code outside `openspec/changes/data-access-consolidation/` was modified.

## 6. Verification

- [ ] 6.1 Run `task pre-push` (gofmt, go test, govulncheck, TypeScript compile, Go binary build) and fix failures.
- [ ] 6.2 Verify rollback: flip one migrated slice back to its raw-SQL binding behind the same interface and confirm handlers still pass.
