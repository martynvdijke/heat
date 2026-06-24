## Context

The project uses Playwright for E2E testing with three browser projects (chromium-desktop, firefox-desktop, mobile-chrome). Tests live in `tests/` and the server starts fresh via `go run .` which uses an in-memory SQLite DB seeded with default data (5 racers, race info, tracks, etc.). The existing test suite covers page load/smoke tests, some admin CRUD, API health checks, and UI accessibility patterns — but there is zero coverage of the actual race gameplay loop.

The key architectural insight is that the server resets to a known state on each run (the `webServer` command in `playwright.config.ts` deletes `heat.db` and starts fresh). This means tests can rely on a deterministic starting state: 5 predefined racers, Monza track, 53 laps, etc.

## Goals / Non-Goals

**Goals:**
- Achieve E2E coverage of the full race lifecycle: start lights → race → rounds → finish → history → stats
- Cover racer CRUD via admin API and verify racer list/leaderboard reflect changes
- Cover player session flow (login, report gear/heat/turbo, status, logout)
- Cover game mechanics (heat cards, upgrades, deck builder)
- Cover flag/lights API commands (trigger, abort, reset)
- Cover round snapshots (create, retrieve by ID, delete)
- Cover weather conditions (CRUD)
- Follow existing patterns: `test.describe` blocks, `loginAdminViaAPI` helper, `API_HEADERS`
- All tests must be hermetic (no shared state between tests beyond what the server seed provides)
- Run in CI alongside existing tests in 3 browsers

**Non-Goals:**
- No unit or integration tests (those live in `*_test.go` files)
- No visual regression testing (screenshots, Percy, etc.)
- No testing of WebSocket real-time updates (covered by Go unit tests)
- No performance or load testing
- No changes to the application code itself
- No mobile-specific race interaction tests (touch drag, etc.)

## Decisions

**Test file organization**: Add new spec files grouped by capability, matching the naming convention of existing files. Use one file per major area.

**Shared helpers**: Leverage existing `loginAdminViaAPI` in `new-features.spec.ts`. The admin login pattern is already established — reuse it rather than creating a new auth helper.

**Sequential vs parallel tests**: The existing `fullyParallel: false` config is correct because tests share a server with seeded data. Serial execution prevents race conditions from state mutations. Specifically:
- Racer CRUD tests must be serial since they modify the racer list
- Race lifecycle tests must be serial since they create/consume race history
- Game mechanics tests that initialize heat decks must be serial

**Data isolation**: Each test should clean up any data it creates where feasible (delete racers, delete race history, etc.). For tests that inherently change global state (e.g., updating race info), use `test.describe.serial` and order tests deliberately.

**Error assertions**: Use the same pattern as existing tests — check `res.ok()` for API calls, check response bodies contain expected data shapes. Use `expect(res.status()).toBe(401)` for auth failure cases.

**No WebSocket assertions**: The real-time broadcast system (WebSocket) is tested via Go unit tests. The E2E tests focus on REST API behavior and page rendering.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Tests flake due to timing (async WebSocket broadcasts affecting API responses) | All assertions are on REST API responses, not WebSocket delivery. API handlers respond synchronously — broadcasts are fire-and-forget via goroutines. |
| Race conditions from parallel tests | Config already uses `fullyParallel: false`. All new tests will be in serial groups. |
| Test data leaking between specs | Each describe block cleans up after itself. Racer CRUD tests delete created racers. Race tests can use unique race names for targeted cleanup. |
| Playwright browser startup time makes test suite slow | The suite already runs in 3 browsers. Keeping individual tests fast (<5s each) and using serial execution minimizes overhead. |
| Admin auth breaks across test files | The `loginAdminViaAPI` helper handles both setup (first-run registration) and login. Each test file that needs admin auth calls it in its first test. |
