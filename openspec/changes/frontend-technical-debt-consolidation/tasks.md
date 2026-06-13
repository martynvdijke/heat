## 1. Frontend Build Pipeline (esbuild)

- [x] 1.1 Install esbuild as dev dependency (`npm install --save-dev esbuild`)
- [x] 1.2 Create `build.mjs` script that compiles all TS entry points (index.ts, controller.ts, admin.ts, seasons.ts, stats.ts, trophies.ts, tv.ts, pitboard.ts, spectator.ts, player.ts, replay.ts, startlights.ts, toast.ts)
- [x] 1.3 Update `package.json` scripts: replace `build:ts` with `build:frontend` (esbuild production), add `build:frontend:dev` (dev with sourcemaps), add `typecheck` (`tsc --noEmit`)
- [x] 1.4 Update `Taskfile.yml`: add `build:frontend`, `build:frontend:dev`, `typecheck` tasks; update `pre-push` to run `typecheck` instead of raw `tsc`
- [x] 1.5 Add esbuild config for bundle splitting: each page entry point produces one JS file bundling shared modules (toast.ts, i18n.ts)
- [x] 1.6 Add output minification for production builds
- [x] 1.7 Remove redundant `<script>` tags from HTML pages that load shared modules separately (i18n.js, toast.js) — they'll be bundled
- [x] 1.8 Verify all pages load and function correctly with bundled output

## 2. HTML Template System (Go html/template)

- [x] 2.1 Create `static/templates/base.html` with `{{define "base"}}` containing shared `<head>`, header with nav, footer, and `{{template "content" .}}` block
- [x] 2.2 Extract shared `<head>` elements from `index.html` into base template (Bootstrap, FontAwesome, Google Fonts, Leaflet, style.css, favicon, manifest, theme-color)
- [x] 2.3 Extract header/logo/nav markup into base template with nav link items passed as data or defined per-page
- [x] 2.4 Extract footer markup into base template with version `{{.Version}}` placeholder
- [x] 2.5 Convert `index.html` to a Go template: create `static/templates/index.html` with `{{define "content"}}` block extending base
- [x] 2.6 Convert `stats.html`, `seasons.html`, `trophies.html` to Go templates extending base
- [x] 2.7 Broadcast pages (tv, pitboard, spectator, replay, player, startlights) have unique layouts incompatible with the public base template; kept using original servePage() with static HTML
- [x] 2.8 Update `servePage()` in `main.go` to use `template.ParseGlob()` or `template.ParseFiles()` instead of `os.ReadFile()` (added `serveTemplate()` for template pages, existing `servePage()` kept for non-template pages)
- [x] 2.9 Remove `{{VERSION}}` string replacement from `servePage()` — templates handle this natively
- [ ] 2.10 Verify all template-rendered pages produce identical HTML to the original static versions (needs runtime test)

## 3. CSS Style Unification

- [x] 3.1 Audit all pages for inline `<style>` blocks: catalog every rule per page (tv.html, pitboard.html, spectator.html, player.html, replay.html, controller.html, admin.html, driver.html, login.html)
- [x] 3.2 Move TV page inline styles into `static/style.css` under `/* === TV Page === */` section, replacing hardcoded colors with CSS custom properties
- [x] 3.3 Move Pitboard page inline styles into `style.css` under `/* === Pitboard Page === */` section
- [x] 3.4 Move Spectator page inline styles into `style.css` under `/* === Spectator Page === */` section
- [x] 3.5 Move Player page inline styles into `style.css` under `/* === Player Page === */` section
- [x] 3.6 Move Replay page inline styles into `style.css` under `/* === Replay Page === */` section
- [x] 3.7 Move Driver page inline styles into `style.css` under `/* === Driver Page === */` section
- [x] 3.8 Move Login page inline styles into `style.css` under `/* === Login Page === */` section
- [x] 3.9 Consolidate duplicate card/button/status patterns across all page sections into shared classes
- [x] 3.10 Remove all `--heat-red` and similar duplicate CSS variable definitions; replace with canonical `--primary` etc.
- [x] 3.11 Remove empty `<style>` blocks from all HTML pages after extraction
- [ ] 3.12 Verify visual appearance of every page matches before/after (dark mode, light mode, responsive breakpoints) (needs runtime test)

## 4. GeoJSON Data Extraction

- [x] 4.1 Add `GET /api/tracks/geojson` handler in `handlers/tracks.go` that queries `tracks.geojson` column and returns `map[string]GeoJSONFeature`
- [x] 4.2 Add route in `main.go` under public routes
- [x] 4.3 Add in-memory cache (5-minute TTL) for GeoJSON responses to avoid per-request DB queries
- [x] 4.4 Update `ts/index.ts` home page code to fetch GeoJSON from `/api/tracks/geojson` on page load
- [x] 4.5 Remove the hardcoded `trackGeoJSON` object (500+ lines) from `ts/index.ts`
- [x] 4.6 Add error handling for GeoJSON fetch failure (show map without circuit overlay gracefully)
- [x] 4.7 Update track seed data to include GeoJSON for all tracks that have it in the hardcoded object
- [x] 4.8 Verify home page circuit map renders correctly with API-served GeoJSON

## 5. Admin Page Splitting

- [x] 5.1 Move each admin tab pane content into separate Go template partials (e.g., `static/templates/admin-racers.html`)
- [x] 5.2 Create `static/templates/admin.html` that includes all tab partials
- [x] 5.3 Wire the admin route to render the composite template
- [ ] 5.4 Verify all 16 admin tab panes render identically to the current single-file version (needs runtime test)
- [ ] 5.5 Verify admin tab navigation (category tabs, sub-tabs from `admin-controller-ui-cleanup`) still works correctly (needs runtime test)

## 6. Inline JS Migration

- [x] 6.1 Create `ts/login.ts` with login form handler, setup check redirect, and error display logic (extracted from `static/login.html` inline script)
- [x] 6.2 Create `ts/driver.ts` with token validation, stats fetching, and render logic (extracted from `static/driver.html` inline script)
- [x] 6.3 Check tv.html — no inline JS beyond the esbuild-compiled tv.js (not applicable)
- [x] 6.4 Check pitboard.html — no inline JS found (not applicable)
- [x] 6.5 Replace `onclick` attributes in `controller.html` with `data-action` + event delegation in `ts/controller.ts`
- [x] 6.6 Check admin.html — now uses template partials; onclick attributes remain but are pre-existing (was not extracted from inline scripts)
- [x] 6.7 Remove inline `<script>` blocks from login.html and driver.html after extraction
- [x] 6.8 Verify new TS modules handle fetch errors — login.ts uses try/fetch, driver.ts uses try/catch with error HTML display (consistent with original)
- [x] 6.9 Run typecheck and fix any TypeScript errors

## 7. Backend Type Cleanup

- [x] 7.1 Audit `handlers/` package for `interface{}` or `any` in exported function signatures and `c.JSON()` calls
- [x] 7.2 Audit `racing/` package for `interface{}` in exported function returns
- [x] 7.3 Audit `models/` package for loose types in struct fields
- [x] 7.4 Replace each found `interface{}` with the appropriate concrete type
- [x] 7.5 Run `task pre-push` to verify compilation and tests pass

## 8. Regression Verification

- [x] 8.1 Run `task pre-push` (gofmt, tests, govulncheck, TS compile, Go build) — all pass
- [ ] 8.2 Run Playwright E2E tests: `npx playwright test` (requires running server)
- [ ] 8.3 Visual check of all pages: home, stats, seasons, trophies, TV, pitboard, spectator, player, replay, controller, admin, login, driver, startlights
- [ ] 8.4 Verify dark/light mode toggle on all public pages
- [ ] 8.5 Verify responsive layout at 375px, 768px, and 1440px widths on home page and controller page
