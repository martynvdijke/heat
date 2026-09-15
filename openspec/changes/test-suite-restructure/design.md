## Context

Heat's Go tests are 25 files at the repo root in `package main` with numeric prefixes (`00_`, `01_`, ... `90_`) that encode a required execution order over a single shared in-memory SQLite DB initialized in `testhelpers_test.go:TestMain` (see `testhelpers_test.go`, `Taskfile.yml` `go test -v ./...`). Business-logic packages (`handlers/`, `db/`, `racing/`, `middleware/`, `app/`, `pkg/`, `ws/`, `models/`) have zero co-located `_test.go` files. The suite is effectively one ordered integration suite with hidden global-state coupling.

## Goals / Non-Goals

**Goals:**

- Isolated, order-independent tests with per-test temp DB setup (`t.TempDir()`).
- Co-located unit tests for pure logic, starting with `racing/` (elo, qualifying, streaks, consistency).
- A reusable, exported harness so both `package main` and subpackages can build a test `app.Server` with an isolated DB without duplicating helpers.
- Slim retained integration/e2e layer; Playwright stays as the cross-browser layer.
- Remove numeric filename prefixes once order independence is achieved.

**Non-Goals:**

- Changing production behavior or public APIs — this is tech-debt / test-architecture only.
- Replacing Playwright e2e or consolidating `task test` / `task test:e2e` invocation.
- Achieving a specific numeric coverage percentage in this change (focus is placement and isolation; coverage grows incrementally).
- Introducing heavy test frameworks or mocking libraries beyond the standard library and existing dependencies.

## Decisions

### D1: Shared harness in a non-test package (e.g. `internal/testutil`) so `package main` and subpackages can both use it

Rationale: helpers in `testhelpers_test.go` are `package main` and unimportable from subpackages; duplicating them per package creates drift. A non-`_test.go` package is importable everywhere and keeps one canonical server/DB factory.
Alternatives: keep helpers in `package main` and use build tags / file copying (rejected — duplication, not importable); put harness in `app/testutil` (rejected — circular dependency risk).

### D2: Per-test isolated DB (temp file via `t.TempDir()` or in-memory sqlite) instead of one shared DB

Rationale: per-test isolation eliminates ordering requirements and inter-test leakage; `t.TempDir()` gives automatic cleanup; `SetMaxOpenConns(1)` pattern from `testhelpers_test.go` is preserved per isolated DB.
Alternatives: single shared DB with transactional rollbacks (rejected — still couples tests, savepoint complexity); Dockerized Postgres (rejected — heavy, sqlite matches prod test path).

### D3: Classify existing tests into unit vs integration and move accordingly

Rationale: pure-logic tests (e.g. elo, qualifying) belong next to the code they cover for discoverability and fast feedback; route-level tests stay as a slim integration layer.
Alternatives: leave all tests at root behind a harness wrapper (rejected — does not solve co-location / discoverability).

### D4: Enable `t.Parallel()` only where isolation is proven

Rationale: parallelism is safe only after per-test DB isolation is verified; enabling it prematurely surfaces flaky failures.
Alternatives: enable globally on day one (rejected — flaky CI); never enable (rejected — leaves speed on the table).

### D5: Coverage focus on pure logic in `racing/` (elo, qualifying, streaks, consistency) which is highly testable

Rationale: `racing/` is deterministic, has no HTTP/DB entanglement, and is the highest-risk calculation surface; it gives the best coverage ROI first.
Alternatives: start with `handlers/` (rejected — more mocking/DB setup, lower ROI per test).

### D6: Keep Playwright e2e as the cross-browser layer

Rationale: Go tests cover server logic and API contracts; Playwright covers real browser/HTMX behavior; they are complementary and already split in `Taskfile.yml` (`go test -v ./...` vs `task test:e2e`).
Alternatives: replace Playwright with Go-only e2e (rejected — loses cross-browser signal); merge into one `go test` run (rejected — different runtimes).

## Risks / Trade-offs

- Moving tests may reveal latent inter-test coupling (good, but expect breakage) → migrate test-by-test behind the new harness; keep old files until new ones pass.
- Ent client setup cost per test (schema migration per isolated DB) → share schema migration helper; measure CI time; consider in-memory sqlite for speed.
- SQLite temp file cleanup / file-descriptor leaks → use `t.TempDir()` for auto-cleanup; close `*sql.DB` and `ent.Client` in `t.Cleanup`.
- CI runtime increase from per-test DB creation → gate `t.Parallel()` where proven; cache Go modules; monitor CI duration.
- Developer churn during migration (two places to look) → document mapping in tasks; remove prefixes only after order independence is proven.

## Migration Plan

1. Introduce harness package (`internal/testutil`) with exported `NewTestServer(t *testing.T) *app.Server` (isolated temp DB, `t.TempDir()`, `t.Cleanup`, single-connection sqlite, Ent migration) without deleting existing tests.
2. Prove one subpackage unit test: add co-located `racing/*_test.go` covering a pure function (e.g. elo or qualifying) using the harness (or no DB at all where possible); verify `go test -v ./...` picks it up.
3. Classify existing root tests into unit vs integration; migrate pure-logic cases to co-located unit tests package by package (`racing/` → `db/` → `handlers/` pure helpers).
4. Refactor remaining integration tests at root to use per-test isolated DB via the harness; add `t.Parallel()` incrementally where isolation is proven.
5. Drop numeric filename prefixes once the suite passes in any order (and with `-shuffle` / `-parallel` where enabled).
6. Add CI gate: `go test -v ./...` must pass shuffled; optional coverage focus report for `racing/`.

Rollback: keep old `NN_test_*.go` files until the replacement co-located / refactored tests pass in CI; if migration regresses, revert to the prefixed files and disable the new harness import — no production code is changed so rollback is test-only.

## Open Questions

- In-memory (`:memory:`) vs temp-file sqlite for per-test DB — in-memory is faster but `t.TempDir()` file gives closer prod parity; decide per package after benchmarking Ent migration cost.
- Whether `package main` tests should stay at repo root or move to a dedicated `app_test` package — affects import cycles and readability; defer until harness proves out.
- Exact coverage gate threshold and whether to enforce it in CI or as an informational report initially.
