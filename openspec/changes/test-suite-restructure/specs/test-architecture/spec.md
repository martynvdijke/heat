## ADDED Requirements

### Requirement: Tests are order-independent and isolated

The test suite SHALL be order-independent: every test that touches persistent state SHALL create its own isolated storage via per-test setup (`t.TempDir()` or equivalent in-memory sqlite) and `t.Cleanup`, with no reliance on execution order or filename-prefix ordering.

#### Scenario: Tests pass in any order

- **WHEN** the Go test suite is run with a shuffled order (e.g. `go test -shuffle=on ./...`)
- **THEN** all tests pass regardless of execution order

#### Scenario: No shared mutable DB between tests

- **WHEN** two tests that write to the database run sequentially or in parallel
- **THEN** neither test observes rows or state written by the other

### Requirement: Business-logic packages have co-located unit tests

Business-logic packages that contain pure logic (notably `racing/`, and incrementally `db/` and `handlers/` pure helpers) SHALL have co-located `_test.go` files exercising that logic without requiring the root `package main` integration harness.

#### Scenario: Racing pure-logic unit test is co-located

- **WHEN** a developer runs `go test ./racing/...`
- **THEN** pure-logic unit tests for `racing/` (e.g. elo, qualifying, streaks, consistency) execute from files co-located in `racing/` and pass without starting the full app server

#### Scenario: Zero co-located tests is a violation

- **WHEN** `racing/` contains testable pure logic
- **THEN** at least one `_test.go` file exists under `racing/` covering that logic (and the pattern extends to `db/` and `handlers/` as migration proceeds)

### Requirement: Shared harness builds a test server with isolated storage

The repository SHALL provide a reusable, exported test harness in a non-`_test.go` package (e.g. `internal/testutil`) that constructs a test `app.Server` with an isolated temp SQLite DB (temp file via `t.TempDir()` or in-memory sqlite), usable from both `package main` and subpackages.

#### Scenario: Harness creates isolated server per test

- **WHEN** a test calls the shared harness helper (e.g. `testutil.NewTestServer(t)`)
- **THEN** it receives an `*app.Server` wired to an isolated SQLite DB (Ent migrated, single-connection where required) that is cleaned up via `t.Cleanup` and does not interfere with other tests

#### Scenario: Harness is importable from subpackages

- **WHEN** a subpackage test (e.g. `racing/elo_test.go`) imports the harness package
- **THEN** the import succeeds and the test can build a test server without importing `package main` or duplicating helpers

### Requirement: Full suite passes in any order and in parallel where declared

The full Go suite invoked via `go test -v ./...` (as run by `Taskfile.yml` `task test`) SHALL pass in any file order and SHALL support `t.Parallel()` for tests whose isolation has been proven.

#### Scenario: Full suite passes via Taskfile entry point

- **WHEN** `go test -v ./...` (or `task test`) is run from the repo root
- **THEN** all tests across `package main` and subpackages pass, including newly added co-located unit tests

#### Scenario: Parallel tests do not flake

- **WHEN** tests marked with `t.Parallel()` run concurrently
- **THEN** they pass reliably because each uses isolated storage and does not share mutable global state

### Requirement: No reliance on filename prefixes for ordering

The test suite SHALL NOT rely on numeric filename prefixes (e.g. `00_`, `01_`, ... `90_`) to enforce execution order; once order independence is achieved, numeric prefixes SHALL be removed.

#### Scenario: Renaming a test file does not change outcome

- **WHEN** a test file is renamed to remove its numeric prefix (e.g. `04_test_race_test.go` → `race_test.go`)
- **THEN** the suite still passes and no test depends on lexicographic file ordering

#### Scenario: No ordering logic in TestMain

- **WHEN** inspecting `TestMain` or global test setup
- **THEN** there is no ordering dependency, shared mutable DB, or file-order assumption that would cause a different file sort order to fail
