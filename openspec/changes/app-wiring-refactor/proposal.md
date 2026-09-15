## Why

`main.go` is 800 lines and `func main()` alone is 605 lines (starts at line 195). Only a handful of functions exist in `main.go` (`loadVendorManifest`, `servePage`, `loadCachedTemplate`, `serveTemplate`, `initAdminTemplate`, `isValidAdminTab`, `main`). `func main()` does all of: env/config loading, DB/Ent setup via `app.Server` (`app/app.go`, 130 lines) and `handlers.Handler` wrapping `*app.Server` (`handlers/handler.go`), websocket manager setup (`ws.NewManager`), ~200 route registrations (`r.GET/POST/PUT/PATCH/DELETE`, `admin := r.Group("/api")`), middleware wiring (`gin.Logger`/`gin.Recovery`, `otelgin`, `gzip` with `/ws` exclusion, `RequestIDMiddleware`, `SecurityHeaders`, `metricsMiddleware`), static/media serving, swagger (`ginSwagger.WrapHandler`, `docs/docs.go` 3594 lines, `docs/swagger.json`/`static/swagger.json` via `task generate-swagger`), service worker (`/sw.js`), sitemap, and HTML page routes spanning auth, oidc, racers, rounds, seasons, stats (many), trmnl, heat-cards/gear-shifts/upgrades, player, spectator, weather, turbo, lap-records, sectors, race-events, commentary, race-radio, i18n, teams, uploads, admin, metrics, and docs. The route surface is unreadable, merge conflicts are frequent, and root tests (`package main`) must boot the whole app because wiring is not reusable.

## What Changes

- Extract route registration from `main()` into grouped, per-domain registration functions (e.g. `registerAuthRoutes`, `registerRacerRoutes`, `registerStatsRoutes`, ...) covering the existing surface (auth/oidc, racers, rounds, seasons, stats, trmnl, heat-cards/gear-shifts/upgrades, player, spectator, weather, turbo, lap-records, sectors, race-events, commentary, race-radio, i18n, teams, uploads, admin, metrics, docs, websocket, HTML pages).
- Extract bootstrap/server construction out of `main()` (env/config, DB/Ent via `app.Server`, `ws.NewManager`, `handlers.New`, logger, cache) into a reusable constructor/initializer.
- Move template loading (`loadCachedTemplate`, `initAdminTemplate`, `loadVendorManifest`, `servePage`/`serveTemplate`) and static/swagger/service-worker wiring out of `main()` into a dedicated web/template/static wiring unit.
- Make `main()` a thin orchestrator that calls bootstrap, middleware setup, route registration, and `r.Run`.
- Preserve exact existing behavior: same paths, HTTP methods, middleware order, handler bindings, and static/swagger semantics; no endpoint changes. Keep `gin` and `app.Server`/`handlers.Handler` boundaries.
- Make wiring callable from tests so the test harness can reuse route/middleware registration without booting the full app.

## Capabilities

### New Capabilities

- `application-wiring`: Grouped, reusable application wiring (bootstrap + per-domain route registration + template/static/swagger wiring) with `main()` as a thin orchestrator, middleware order explicit and preserved, and route surface enumerable/verifiable with no behavior change to existing endpoints.

### Modified Capabilities

None.

## Impact

Refactor-only, behavior-preserving. `main.go` shrinks to an orchestrator; new wiring units (e.g. `routes/` or builder methods — see design) own domain-grouped registration. `app/app.go` and `handlers/handler.go` boundaries unchanged. Generated swagger (`docs/docs.go`, `docs/swagger.json`, `static/swagger.json`) and `task generate-swagger` unchanged. Risk is accidental route/middleware-order drift (mitigated by route-inventory snapshot test); migration is batched by domain with snapshot verification and per-batch revert.
