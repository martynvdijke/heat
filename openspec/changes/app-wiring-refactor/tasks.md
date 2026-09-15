## 1. Snapshot and scaffolding

- [ ] 1.1 Add a route-inventory snapshot test that enumerates the current Gin route table (method + path + handler/middleware identity) and commits the golden file; test fails on any drift.
- [ ] 1.2 Create the wiring destination package (e.g. `routes/` or `internal/routes/`) and decide on free functions vs builder methods per design D2; add package doc and keep `gin`/`app.Server`/`handlers.Handler` boundaries.
- [ ] 1.3 Extract bootstrap/server construction out of `main()` into a reusable constructor/initializer (env/config, DB/Ent, `ws.NewManager`, `handlers.New`, logger/cache) callable from tests; keep `main()` as caller and verify snapshot unchanged.

## 2. Template, static, and swagger wiring

- [ ] 2.1 Move `loadVendorManifest`, `loadCachedTemplate`, `initAdminTemplate`, `servePage`/`serveTemplate` out of `main.go` into a dedicated web/template unit; preserve caching and vendor manifest semantics.
- [ ] 2.2 Move static/media (`/static`, `/media`), swagger (`/swagger/*`, `/api-docs`, `/docs`), service worker (`/sw.js`), and cache-header wiring out of `main()` into wiring; verify `docs/docs.go` (3594 lines) and `task generate-swagger` remain unchanged.

## 3. Route registration — public and auth

- [ ] 3.1 Extract websocket (`GET /ws`), auth/password (`POST /api/login`, `POST /api/logout`, `/api/check-setup`, forgot/reset), and OIDC (`/api/auth/oidc/*`) registrations into `registerWebSocketRoutes`/`registerAuthRoutes`.
- [ ] 3.2 Extract public read routes (`GET /api/racers`, `/api/uploads`, `/api/race-info`, `/api/tracks*`, `/api/race-history`, `/api/racer-stats*`, `/api/quotes*`, `/api/teams*`, version/metrics/docs) and verify snapshot after each batch.

## 4. Route registration — racing and stats domains

- [ ] 4.1 Extract racers/rounds/seasons/tracks (`admin.POST/PUT/DELETE /api/racers*`, `/api/race-info`, `/api/tracks*`, `/api/rounds*`, `/api/seasons*`) and TRMNL (`/api/trmnl/*`) into domain functions.
- [ ] 4.2 Extract stats routes (`/api/stats/*`, `/api/track-stats`, `/api/oneoff-races`) and HTMX admin fragments under `admin.Group("/api")` into grouped functions.
- [ ] 4.3 Extract heat-cards/gear-shifts/upgrades and game-mechanics routes (`/api/heat-cards*`, `/api/gear-shifts*`, `/api/upgrade-cards*`, `/api/player-upgrades*`, `/api/legend-abilities*`) into a game-mechanics group.

## 5. Route registration — player, spectator, and race enhancements

- [ ] 5.1 Extract player/spectator/multi-user routes (`/api/player/*`, `/api/spectator/state`, `/api/player-sessions`) and teams (`/api/teams*`) into domain functions.
- [ ] 5.2 Extract race-enhancement routes (weather, turbo-logs, lap-records, sectors, race-events, ai-difficulty, sound, race-radio, commentary, flags) and i18n/driver-share/uploads into domain functions.
- [ ] 5.3 Extract admin HTML pages and HTMX endpoints (`/admin.html`, `/login.html`, `/setup`, `/controller.html`, `/stats.html`, `/seasons.html`, `/tv.html`, `/pitboard.html`, `/replay.html`, `/player.html`, `/spectator.html`, `/driver.html`, `/`, and `admin.GET /api/html/*`) with `UmamiMiddleware`/`I18nMiddleware` ordering preserved.

## 6. Middleware ordering and verification

- [ ] 6.1 Make global and per-group middleware order explicit and documented (gin.Logger/Recovery, otelgin, gzip with `/ws` exclusion, RequestIDMiddleware, SecurityHeaders, metricsMiddleware; admin `CSRFMiddleware`+`AuthMiddleware`) and add a test/comment asserting the order.
- [ ] 6.2 Run `task pre-push` (gofmt, `go test ./...`, govulncheck, TS build) and `task generate-swagger` verification; confirm route-inventory snapshot unchanged; remove dead code from `main.go` so `main()` is a thin orchestrator and wiring is reusable from tests.
