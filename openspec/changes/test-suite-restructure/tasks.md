## 1. Harness — shared isolated test server

- [ ] 1.1 Create `internal/testutil` (non-`_test.go` package) with exported `NewTestServer(t *testing.T) *app.Server` that creates an isolated temp SQLite DB via `t.TempDir()` (or `:memory:`) with `SetMaxOpenConns(1)`, Ent client + schema migration, `t.Cleanup` DB close, and minimal `app.Server` wiring (BasePath, MediaPath, Broadcast stubs, logger) — importable from `package main` and subpackages.
- [ ] 1.2 Add `t.TempDir()` / `t.Cleanup` helpers for DB file lifecycle and verify no file-descriptor leaks; document in-memory vs temp-file choice.
- [ ] 1.3 Update `Taskfile.yml` docs/comments if needed; confirm `go test -v ./...` still discovers the new package.

## 2. Prove co-located unit tests in racing/

- [ ] 2.1 Add `racing/*_test.go` covering pure logic (elo, qualifying, streaks, or consistency) using the new harness where DB is needed (or no harness where pure).
- [ ] 2.2 Verify `go test ./racing/...` and `go test -v ./...` both pass; ensure existing root tests still pass unmodified.
- [ ] 2.3 Extend co-located tests to remaining pure `racing/` logic; add coverage focus report for `racing/`.

## 3. Classify and migrate existing root tests

- [ ] 3.1 Audit 25 root test files (plus `paths_test.go`, `testhelpers_test.go`) and classify each `func Test...` as unit vs integration; record mapping.
- [ ] 3.2 Migrate pure-logic cases to co-located unit tests (`db/` helpers, `handlers/` pure helpers) using the shared harness.
- [ ] 3.3 Refactor remaining `package main` integration tests to use per-test isolated DB via the harness (replace shared `TestMain` DB with per-test `NewTestServer(t)`); keep a slim end-to-end layer.

## 4. Decouple ordering and enable parallelism

- [ ] 4.1 Remove reliance on numeric filename prefixes: verify suite passes with `go test -shuffle=on ./...`; then rename files to drop prefixes incrementally.
- [ ] 4.2 Enable `t.Parallel()` only for tests with proven isolation; verify no flaky failures in CI.
- [ ] 4.3 Migrate shared helpers from `testhelpers_test.go` into `internal/testutil` (or re-export) and delete duplication.

## 5. CI gate and verification

- [ ] 5.1 Add CI check that `go test -v ./...` passes shuffled and that `task test` + co-located tests all run; keep `task test:e2e` (Playwright) as the separate cross-browser layer unchanged.
- [ ] 5.2 Add optional coverage focus report for `racing/` (and later `db/`/`handlers/`); decide on gate vs informational.
- [ ] 5.3 Final verification: fresh checkout, `go test -v ./...` and `go test -shuffle=on ./...` pass; no test depends on file order; file list reflects new co-located tests and removed prefixes.
