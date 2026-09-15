## Context

`main.go` is 800 lines; `func main()` alone is 605 lines starting at line 195. Only seven functions exist in the file: `loadVendorManifest`, `servePage`, `loadCachedTemplate`, `serveTemplate`, `initAdminTemplate`, `isValidAdminTab`, and `main`. `func main()` currently does everything: env/config loading, `app.Server` construction (`app/app.go`, 130 lines) and `handlers.Handler` wrapping `*app.Server` (`handlers/handler.go`), DB/Ent (`sql.Open` + `entsql.OpenDB`), websocket manager (`ws.NewManager` + broadcast goroutines), middleware wiring (`gin.Logger`/`gin.Recovery`, `otelgin.Middleware`, `gzip` with `/ws` exclusion, `RequestIDMiddleware`, `SecurityHeaders`, `metricsMiddleware`), ~200 route registrations (`r.GET/POST/PUT/PATCH/DELETE`, `admin := r.Group("/api")` with `CSRFMiddleware` + `AuthMiddleware`), static/media (`/static`, `/media`), swagger (`ginSwagger.WrapHandler`, `docs/docs.go` 3594 lines, `docs/swagger.json`/`static/swagger.json` via `task generate-swagger`), service worker (`/sw.js`), and HTML page routes. Domains span auth, oidc, racers, rounds, seasons, stats (many), trmnl, heat-cards/gear-shifts/upgrades, player, spectator, weather, turbo, lap-records, sectors, race-events, commentary, race-radio, i18n, teams, uploads, admin, metrics, and docs. Root tests (`package main`) must boot the whole app because wiring is not reusable.

## Goals / Non-Goals

**Goals:**
- Make `main()` a thin orchestrator; route surface becomes readable and diffable per domain.
- Group route registration by domain/capability (e.g. `registerAuthRoutes`, `registerRacerRoutes`, `registerStatsRoutes`) so ownership and merge conflicts are isolated.
- Extract bootstrap/server construction and template/static/swagger wiring out of `main()`.
- Keep `gin`, `app.Server`, and `handlers.Handler` boundaries; no framework change.
- Preserve behavior exactly: same paths, methods, middleware order, handler bindings.
- Make wiring reusable from tests so `package main` tests can reuse it without booting the full app.
- Keep middleware ordering explicit and documented (security depends on CSRF/Auth/SecurityHeaders order).

**Non-Goals:**
- Changing any route path, method, auth requirement, or handler behavior.
- Replacing `gin` or introducing a new router/framework.
- Changing `app.Server`/`handlers.Handler` ownership or Ent/DB layer.
- Regenerating or restructuring swagger generation (`task generate-swagger`, `docs/docs.go`).
- Adding new endpoints or middleware behavior.

## Decisions

### D1: Grouped functions by domain vs declarative route table

Choose grouped functions (e.g. `registerAuthRoutes(r, h, s)`, `registerRacerRoutes(...)`, `registerStatsRoutes(...)`) over a single declarative route table (slice/map of `{method, path, handler, middleware}`).

Rationale: typed handler references give compiler-checked diffs, per-domain files isolate merge conflicts, middleware composition stays as ordinary Go code, and `gin.Group` usage remains idiomatic. A declarative table centralizes the inventory but adds an abstraction layer, loses type safety on handler signatures, and makes per-route middleware composition harder to read.

Alternative: a route table helps when you need code-generated docs or uniform policy enforcement across all routes; revisit if a generated route inventory or policy linter is required.

### D2: Where registration lives — new package vs methods on a router builder

Prefer a new package (e.g. `routes/` or `internal/routes/`) with free functions `RegisterXxxRoutes(...)` that accept `*gin.Engine`/`gin.RouterGroup`, `*handlers.Handler`, and `*app.Server`, over methods on a custom router-builder type.

Rationale: a package cleanly separates wiring from `main` without introducing a builder abstraction that would be the only implementor; free functions are the smallest seam that lets tests import and call wiring directly. A builder type centralizes state but adds an interface with one implementation and obscures plain `gin` usage.

Alternative: methods on a `Router`/`App` builder if registration needs shared mutable state or fluent chaining; adopt only if cross-domain ordering or conditional registration becomes complex.

### D3: No behavior change — same paths, methods, middleware order

Any refactor SHALL preserve the exact external contract: identical paths, HTTP methods, middleware stack and order, and handler bindings. No endpoint is added, removed, or re-pathed.

Rationale: this is a pure wiring refactor; behavior changes would conflate review and risk. Alternatives (renaming, consolidating, or re-middling routes) are deferred to follow-up changes.

### D4: Wiring callable from tests

Extracted bootstrap and registration functions SHALL be importable/callable from tests so the test harness can reuse wiring without booting the full app via `main()`.

Rationale: current `package main` tests must boot the whole app; reusable wiring enables focused tests and a route-inventory snapshot test. Alternative (keeping wiring private to `main`) preserves encapsulation but keeps tests coupled to the god function.

### D5: Keep gin

Keep `gin-gonic/gin` as the HTTP framework; no migration to `net/http`, `chi`, or `echo`.

Rationale: all handlers, middleware (`gin.Logger`/`gin.Recovery`, `otelgin`, `gzip`, `RequestIDMiddleware`, `SecurityHeaders`), groups (`admin := r.Group("/api")`), and swagger (`ginSwagger.WrapHandler`) are gin-coupled. Framework churn adds risk with no wiring benefit.

### D6: Keep middleware ordering explicit and documented

Middleware order SHALL remain explicit in code and documented because security (CSRF/Auth/SecurityHeaders, rate limiting, gzip `/ws` exclusion, metrics) depends on it.

Rationale: reordering `CSRFMiddleware`/`AuthMiddleware`/`SecurityHeaders` or `gzip` exclusion can cause auth bypass or CSRF gaps and break websocket upgrades. Explicit, commented ordering is cheaper and safer than implicit builder ordering.

## Risks / Trade-offs

- Accidental route or middleware-order change causes auth bypass or CSRF gaps → mitigate with a route-inventory snapshot test that fails on any path/method/middleware drift; review diffs per domain batch.
- PR noise and churn across `main.go` → mitigate by migrating in small domain batches, each verified against the snapshot; keep commits reviewable.
- Duplicated registration during migration (old + new wiring both reachable) → mitigate by removing old registrations in the same commit as the new ones; snapshot test catches duplicates.
- Template/static/swagger wiring drift (cache headers, vendor manifest, `/sw.js` semantics) → mitigate by moving wiring verbatim first, then verifying static/swagger tests and manual smoke.

## Migration Plan

1. Snapshot the current route table in a test (enumerate `r.Routes()` or equivalent) and commit the inventory as a golden file / snapshot assertion.
2. Extract bootstrap/server construction out of `main()` into a reusable constructor; keep `main()` calling it — verify snapshot and tests unchanged.
3. Extract template loading and static/swagger/service-worker wiring out of `main()` — verify snapshot unchanged.
4. Move routes in domain batches (e.g. auth/oidc → racers → seasons → stats → trmnl → heat-cards/gear-shifts/upgrades → player/spectator → weather/turbo/lap-records/sectors/race-events/commentary/race-radio → i18n/teams/uploads → admin/metrics/docs/pages); after each batch verify the route-inventory snapshot is unchanged.
5. Remove dead code from `main()` and ensure `main()` is a thin orchestrator (bootstrap → middleware → registration → `r.Run`).
6. Delete any shims; run `task pre-push` (gofmt, `go test ./...`, `govulncheck`, TS build) and `task generate-swagger` verification.

Rollback: revert per-batch commits in reverse order; snapshot test identifies which batch introduced drift.

## Open Questions

- What package layout should registration live in (`routes/`, `internal/routes/`, `app/routes/`, or `internal/http/`)?
- Should `servePage`/`loadCachedTemplate`/`initAdminTemplate`/`loadVendorManifest` move to a `web/` or `templates/` package, or stay alongside wiring?
- Should the route inventory become a generated doc (e.g. `docs/routes.md` or `static/routes.json`) in addition to the snapshot test?
