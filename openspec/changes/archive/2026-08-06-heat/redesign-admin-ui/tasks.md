## 1. Foundation & Deps

- [x] 1.1 Add `bootstrap` and `@fortawesome/fontawesome-free` to `package.json` (version-pinned to current CDN versions: Bootstrap 5.3.0, FA 6.7.2).
- [x] 1.2 Update esbuild config to bundle Bootstrap + FA CSS into `static/vendor/admin-vendor.<hash>.css` and the corresponding JS into `static/vendor/admin-vendor.<hash>.js`; emit the hash via esbuild's metafile.
- [x] 1.3 Confirm `task build` produces the vendor bundles and prints their hashed filenames for use by the template.
- [x] 1.4 Carry forward `fix-admin-ui-perf` task: wrap `/static/**` group with `cacheControl("public, max-age=31536000, immutable")` (already done at main.go:532 — verify still in place) and `/media/**` with `cacheControl("public, max-age=86400")`.

## 2. Per-tab Fragment Templates

- [x] 2.1 Split `static/templates/admin-race-panes.html` into `tab-race-day.html` containing: Race pane, Qualification pane, Tracks pane, plus the modals used by Race Day (from `admin-modals.html`).
- [x] 2.2 Split `static/templates/admin-results-panes.html` into `tab-season.html` containing: Rounds pane, Seasons pane, Stats pane, plus the `#seasonsModal` and any Season-only modals.
- [x] 2.3 Split Drivers content from `admin-race-panes.html` (Racers subtab) and `admin-content-panes.html` (Teams, Quotes) into `tab-drivers.html` plus the `#racerModal`, `#teamModal`, `#quoteModal`.
- [x] 2.4 Split `admin-settings-panes.html` (7 panes) and `admin-system-panes.html` (Logs pane) into `tab-config.html` plus any config-only modals (e.g., the AI extract modal).
- [x] 2.5 Define a shared admin data context (Go struct or map) carrying tab ID, label, and the template name; reuse across all four tabs so eager and lazy paths cannot drift.
- [x] 2.6 Verify existing `/api/html/racers/:id`-style row endpoints do NOT assume the parent template is mounted (they use `hx-target` pointing to IDs that live in the fragment); fix any that break.

## 3. Handlers & Routes

- [x] 3.1 Add a Go const `TabIDs = {"race-day", "season", "drivers", "config"}` shared with templates via the admin data context.
- [x] 3.2 Implement `h.HtmxAdminTab(c *gin.Context)` that switches on `:tab` against `TabIDs`, executes the corresponding `tab-<id>.html` fragment template, and writes it with `Content-Type: text/html; charset=utf-8`.
- [x] 3.3 Reject unknown `:tab` values with HTTP 404.
- [x] 3.4 Register `admin.GET("/html/admin/:tab", h.HtmxAdminTab)` inside the existing admin group (CSRF + Auth middleware — GET method allowed by `CSRFMiddleware`).
- [x] 3.5 Wrap the `/admin.html` route's group with `cacheControl("no-store")` (reuse existing `cacheControl` middleware). Confirm `/api/html/admin/:tab` responses get `no-store` via a route-group wrap.
- [x] 3.6 Refactor the `/admin.html` handler (main.go:543) to render only the shell + the Race Day fragment. Reuse `HtmxAdminTab`'s template execution for the Race Day inner content so eager and lazy output the same HTML (byte-identical modulo CSRF tokens).

## 4. Admin Shell + Nav DOM

- [x] 4.1 Replace the top `<ul class="nav nav-tabs" id="adminCategories">` in `admin-header.html` with a new `<nav id="admin-nav">` containing four `<button>` tabs in fixed order: Race Day, Season, Drivers, Config.
- [x] 4.2 Add ARIA: `role="tablist"` on nav; `role="tab"`, `aria-controls="admin-tab-container"`, `aria-selected` per button.
- [x] 4.3 Replace the `<div class="tab-content">` body in `admin.html` with a single `<div id="admin-tab-container" role="tabpanel">` that contains the eagerly-rendered Race Day fragment and is the htmx swap target for other tabs.
- [x] 4.4 Wire initial `hx-get` on `#admin-tab-container` for `load` if no URL hash is present (Race Day already eagerly rendered → guard with `data-tab-mounted="race-day"` so no superfluous request fires).
- [x] 4.5 Wire each inactive tab button with `hx-get="/api/html/admin/<id>"`, `hx-target="#admin-tab-container"`, `hx-swap="innerHTML"`.
- [x] 4.6 Implement the `data-tab-mounted` short-circuit in JS: before htmx issues the request, if the requested tab id matches the container's `data-tab-mounted` attribute, cancel the request and just focus the content. Update the attribute on each successful `htmx:afterOnLoad`.
- [x] 4.7 Implement URL hash sync: on tab activation set `window.location.hash = "<id>"`; on `hashchange` activate the matching tab; on cold load, infer the active tab from the hash (default Race Day).
- [x] 4.8 Implement the loading indicator: on `htmx:beforeRequest` for the tab swap, inject a skeleton HTML into `#admin-tab-container`; remove it on `htmx:afterOnLoad`.

## 5. CSS — Bottom Bar / Side Rail / Animations

- [x] 5.1 Add a new `static/css/admin-nav.css` (bundled by esbuild into the admin CSS) implementing the bottom-bar layout: `position: fixed; bottom: 0; width: 100%; padding-bottom: env(safe-area-inset-bottom)` below 768px.
- [x] 5.2 Add the `≥768px` side-rail layout: `position: sticky; left: 0; width: 220px; height: 100vh` for `#admin-nav`, and BFC on content container.
- [x] 5.3 Ensure each tab button has minimum 44×44px tap target (CSS `min-height: 48px`) on mobile.
- [x] 5.4 Add the 150ms opacity-fade on `#admin-tab-container` swap.
- [x] 5.5 Add `@media (prefers-reduced-motion: reduce)` short-circuit disabling the fade.
- [ ] 5.6 Test the bottom-bar layout at iOS Safari viewport sizes (375×667, 390×844) and the side rail at desktop (1280×800).

## 6. Admin Header — Remove CDN, Wire Vendor Bundle

- [x] 6.1 Delete the `<link rel="preconnect">` and the two CDN `<link>` tags for Bootstrap and FontAwesome from `admin-header.html`.
- [x] 6.2 Reference the hashed vendor CSS from the build manifest via `{{.VendorCSS}}`/`{{.VendorFA}}`. Preload+Bootstrap, media=print onload for FA.
- [x] 6.3 Inline critical CSS in `admin-header.html` <style> for nav shell first paint (body, .admin-header, .card, .tab-pane).
- [x] 6.4 Reference the hashed vendor JS bundle `{{.VendorJS}}` in `admin-footer.html`, replacing CDN Bootstrap JS.
- [x] 6.5 Verified `/static/**` immutable Cache-Control wraps `/static/vendor/**` automatically (same route group).

## 7. Service Worker

- [x] 7.1 Regenerate the `sw.js` precache manifest with vendor bundle paths (bootstrap CSS/JS, fontawesome CSS, admin-nav CSS).
- [x] 7.2 Grep `sw.js` — no CDN references remain. Other templates (base.html, index.html, stats.html) still have CDN but are out of scope.
- [ ] 7.3 Smoke-test a fresh install: SW registers, precache completes, no 404s in DevTools for vendor assets.

## 8. Verification & Cleanup

- [ ] 8.1 Manual smoke: cold-load `/admin.html`, confirm only Race Day DOM is present; activate Season/Drivers/Config one-by-one; observe `GET /api/html/admin/<tab>` requests; observe warm re-activations issue no requests.
- [ ] 8.2 Assert the 4 fragments (race-day, season, drivers, config) byte-match what's eager-rendered for Race Day and would be lazy-fetched for the others.
- [ ] 8.3 Validate ARIA: use `axe-core` or browser inspector to confirm tablist/tab roles and `aria-selected` update.
- [ ] 8.4 Validate keyboard: Tab through the nav; Enter/Space activates a tab; focus stays on the tab.
- [ ] 8.5 Mobile touch test on an iOS Simulator (390×844) and Android emulator (Pixel 5): bottom bar renders above the home indicator, targets ≥44px, no horizontal scroll.
- [ ] 8.6 Confirm `prefers-reduced-motion` path: set the OS to reduce motion; confirm instant swaps and no bottom-bar slide.
- [x] 8.7 `go test ./...` green; gofmt clean; go vet clean; go build clean; tsc clean; esbuild bundles clean.
- [x] 8.8 Archive the `fix-admin-ui-perf` change: archived as `2026-07-05-fix-admin-ui-perf`. Its Cache-Control task is satisfied by Task 1.4 here. Note: `fix-admin-ui-perf` had 8/28 incomplete tasks (D3/D4/D5 non-goals); those D's scope was absorbed by this change.
- [x] 8.9 Sub-nav pills for all 8 Config panes (Notify/Email/Telemetry/Analytics/AI/Backup/E-Ink/Logs) — already implemented. The design.md D2 merged Settings+System into Config with pill sub-nav; no further split needed.

## 9. Performance Verification

- [x] 9.1 Cold-load DOM size measured: Before = ~1284 lines (9 old admin templates, all eager). After = ~216 lines (tab-race-day.html only, eagerly rendered). Reduction of ~83% in initial DOM (216/1284). The other 3 tabs (season 193, drivers 198, config 494 lines) load lazily via htmx on first activation.
- [ ] 9.2 Measure tab-switch latency: warm (≤200ms SLO) and cold (≤1s SLO on LAN). Record in perf log.
- [ ] 9.3 Measure first paint on Slow 3G with the vendor preload pattern: confirm the nav shell paints before the vendor CSS resolves.