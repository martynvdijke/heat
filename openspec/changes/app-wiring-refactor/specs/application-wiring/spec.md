## ADDED Requirements

### Requirement: Thin orchestrator main

`main()` SHALL be a thin orchestrator that delegates to extracted bootstrap, middleware, and route-registration units and MUST NOT contain inline route registrations, middleware wiring, or template/static/swagger setup beyond orchestrator calls.

#### Scenario: Main delegates to wiring

- **WHEN** `main()` starts
- **THEN** it calls extracted bootstrap/server-construction, middleware setup, and grouped route-registration functions and then starts the server, with no inline `r.GET/POST/PUT/PATCH/DELETE` or `r.Group` registrations remaining in `main()`.

### Requirement: Grouped route registration by domain

Route registration SHALL be grouped by domain/capability into per-domain functions (e.g. `registerAuthRoutes`, `registerRacerRoutes`, `registerStatsRoutes`, and equivalents for seasons, stats, trmnl, heat-cards/gear-shifts/upgrades, player, spectator, weather, turbo, lap-records, sectors, race-events, commentary, race-radio, i18n, teams, uploads, admin, metrics, docs, websocket, HTML pages) rather than a single monolithic block in `main()`.

#### Scenario: Domain-grouped registration

- **WHEN** a developer needs to find or change racer or stats routes
- **THEN** they locate a single domain-grouped registration function (e.g. racer or stats) that owns those routes without scanning `main()`.

### Requirement: Preserved middleware order

The application SHALL preserve the existing global and per-group middleware order and make that order explicit and documented, because security (CSRF/Auth/SecurityHeaders, rate limiting, gzip `/ws` exclusion) depends on it.

#### Scenario: Middleware order preserved

- **WHEN** the application boots
- **THEN** global middleware is applied in the existing order (`gin.Logger`/`gin.Recovery`, `otelgin`, `gzip` excluding `/ws`, `RequestIDMiddleware`, `SecurityHeaders`, `metricsMiddleware`) and admin groups use `CSRFMiddleware` + `AuthMiddleware` as before, with the order visible and commented in wiring code.

### Requirement: Enumerable and verifiable route surface

The complete route surface (paths, HTTP methods, and middleware associations) SHALL be enumerable and verifiable by a test (e.g. a route-inventory snapshot test) that fails on any drift.

#### Scenario: Route inventory snapshot

- **WHEN** the route table changes (path, method, or middleware added/removed/reordered)
- **THEN** the inventory test fails, requiring an intentional snapshot update.

### Requirement: Reusable wiring for tests

Extracted bootstrap and route-registration wiring SHALL be importable and callable from tests so the test harness can reuse wiring without booting the full application via `main()`.

#### Scenario: Test reuses wiring

- **WHEN** a test needs an HTTP router with the real route/middleware wiring
- **THEN** it can call the extracted registration functions (with `*app.Server`/`*handlers.Handler`) to build the router without executing `main()`.

### Requirement: No behavior change to existing endpoints

The refactor SHALL NOT change existing endpoint behavior: paths, HTTP methods, auth requirements, handler bindings, static/media/swagger/service-worker semantics, and template rendering MUST remain identical.

#### Scenario: Existing endpoints unchanged

- **WHEN** a client calls any existing endpoint (e.g. `/api/racers`, `/api/race-info`, `/api/stats/*`, `/ws`, `/static/*`, `/swagger/*`, HTML pages)
- **THEN** the response status, headers, and semantics are identical before and after the refactor.
