## Why

25 Go test files sit at the repo root, all declared `package main`, with numeric filename prefixes that force execution order: `00_test_auth_test.go`, `00_test_password_reset_test.go`, `01_test_unit_test.go`, `02_test_racers_test.go`, `04_test_race_test.go`, `05_test_stats_test.go`, `06_test_tracks_quotes_teams_test.go`, `07_test_settings_seasons_test.go`, `08_test_api_test.go`, `09_test_infrastructure_test.go`, `10_test_otel_test.go`, `11_test_extensions_test.go`, `12_test_trmnl_next_race_test.go`, `12_test_trmnl_test.go`, `13_test_tracks_modules_test.go`, `14_test_admin_season_stats_test.go`, `15_test_commentary_test.go`, `15_test_extensions_owned_test.go`, `16_test_weather_test.go`, `17_test_stats_season_test.go`, `18_test_oidc_test.go`, `30_test_crud_test.go`, `90_test_htmx_test.go`, plus `paths_test.go` and `testhelpers_test.go` (148 `func Test...` total; largest: 90_test_htmx_test.go (25), 08_test_api_test.go (17), 04_test_race_test.go (16)). There are ZERO `_test.go` files under handlers/, db/, racing/, middleware/, app/, pkg/, ws/, models/ — business logic has no co-located unit tests. `Taskfile.yml` runs `go test -v ./...` and Playwright e2e separately (`task test:e2e`); `testhelpers_test.go` holds shared helpers with a `TestMain` that builds a single shared in-memory SQLite DB and `app.Server`. Root tests exercise the whole app (routes, server) — they are integration tests in `package main`; the numeric ordering encodes shared global state / shared sqlite db. As a result the suite can only run as one ordered integration suite with no isolation, unnumbered business packages are untested, and refactoring is scary.

## What Changes

- Add a reusable, exported test harness that constructs a test `app.Server` with an isolated temp SQLite DB (per-test `t.TempDir()` or in-memory sqlite) so non-`main` packages can be tested without the root `TestMain` global.
- Add co-located unit tests for `racing/`, `db/`, `handlers/` pure logic (starting with `racing/` — elo, qualifying, streaks, consistency — as the highest-value, pure-logic target).
- Decouple tests from filename ordering: per-test setup + `t.TempDir()` isolated DB, no reliance on numeric prefixes; keep a slim end-to-end layer for route-level checks.
- Remove numeric prefixes once order-independence is proven.
- Classify existing tests into unit vs integration and move them accordingly; enable `t.Parallel()` only where isolation is proven; keep Playwright e2e as the cross-browser layer.

## Capabilities

### New Capabilities

- `test-architecture`: Order-independent, isolated test architecture with a shared harness (`internal/testutil` or equivalent non-test package) that builds an isolated test server/DB, co-located unit tests for business-logic packages, and a slim integration/e2e layer without filename-prefix ordering.

### Modified Capabilities

None.

## Impact

- New package: `internal/testutil` (or equivalent) — exported helpers to build `app.Server` with isolated temp SQLite DB, usable from `package main` and subpackages.
- New co-located tests: `racing/*_test.go`, `db/*_test.go`, `handlers/*_test.go` (pure-logic coverage).
- Modified: existing root `*_test.go` files progressively refactored to use per-test setup and `t.TempDir()`; numeric prefixes removed incrementally; `testhelpers_test.go` helpers migrated into the shared harness.
- `Taskfile.yml` / CI: add coverage focus and (eventually) CI gate; `go test -v ./...` remains the single Go test entry point; `task test:e2e` unchanged as the cross-browser layer.
- No production code behavior change (tech-debt / test-architecture only).
