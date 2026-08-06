## Why

The admin page is sluggish on touch devices and slow to paint: the top Bootstrap nav is thumb-hostile (wraps and horizontal-scrolls on phones), and the entire 1284-line DOM — 5 categories × all subtabs × all modals — is rendered upfront, causing the ~10s animation/slow paint when switching tabs that `fix-admin-ui-perf` already flagged. A pure visual regrouping would only fix half the problem; this change rebuilds both the navigation topology and the pane-mounting model so the admin becomes responsive on a touch hand and fast on a cold load.

## What Changes

- **Replace the top Bootstrap `.nav-tabs` with a bottom tab bar** optimised for touch (≥44px targets, `env(safe-area-inset-bottom)`), grouped by admin workflow into **4 tabs**: Race Day, Season, Drivers, Config. On wide screens (≥768px) the same nav is lifted to a side rail (one shared DOM, media-query reflow) to preserve desktop ergonomics.
- **Regroup admin content into the 4 workflow tabs**:
  - **Race Day**: Race, Qualification, Tracks (live operational controls stay here).
  - **Season**: Rounds, Seasons, Stats (results/championship promotion here).
  - **Drivers**: Racers, Teams, Quotes (people & their content).
  - **Config**: all 7 settings panes + System Logs (the boundary between "configure" and "debug" is collapsed into one config tab, keeping the bar at 4).
- **Lazy-mount panes via htmx**: the admin shell renders only the active tab's panes server-side; switching a tab issues `hx-get` to `/api/html/admin/<tab>` to mount that tab's panes on demand. The full sub-DOM is no longer shipped upfront, ending the 10s paint regression at its root.
- **Self-host Bootstrap & FontAwesome CSS** (vendored under `/static/vendor/`) and bundle them with esbuild so the admin shell stops making render-blocking CDN round-trips and has zero external dependencies at first paint. (Promoted from a non-goal of `fix-admin-ui-perf` to a goal here because the redesign already touches the head template.)
- **Add pane-target endpoints** `GET /api/html/admin/{tab}` (server-side fragments) returning only the requested tab's panes + modals, scoped to that tab, with `Cache-Control: no-store` since the content is dynamic.
- **Add a transient loading indicator** surfaced via htmx's `htmx:afterOnLoad`/`htmx:beforeRequest` events so a tab switch shows a skeleton/spinner while the pane mounts (≤200ms typical; visible feedback otherwise).
- **Add a `prefers-reduced-motion` short-circuit** that disables the `.tab-pane` fade and any bottom-bar animations for users who opt out.
- **Deprecate & archive the existing `fix-admin-ui-perf` change**: its CSS-preload and Cache-Control goals are subsumed (self-host removes the CDN preload; the static-route Cache-Control task remains valid and is carried into this change as a smaller task). Its lazy-mount non-goal becomes this change's core mechanism.

## Capabilities

### New Capabilities
- `admin-bottom-nav`: Touch-first bottom tab bar (4 workflow tabs) with safe-area-aware positioning, responsive lift to a side rail ≥768px, ≥44px targets, keyboard-accessible ARIA tablist semantics, and `prefers-reduced-motion` support.
- `admin-lazy-panes`: Server-rendered admin shell mounts only the active tab's panes on load; non-active tabs fetch their panes via htmx `hx-get` to `/api/html/admin/{tab}` on first activation. Panes are cached client-side after first mount so reactivating a tab is instant.
- `admin-vendored-assets`: Bootstrap and FontAwesome CSS+JS are vendored under `/static/vendor/` and bundled by esbuild, eliminating render-blocking CDN round-trips and external origin dependencies from the admin first paint.
- `admin-tab-endpoints`: `GET /api/html/admin/{tab}` endpoint family that returns the HTML fragment for one tab's panes and modals only, scoped to that tab, with `Cache-Control: no-store` and admin auth enforced.

### Modified Capabilities
_None — `openspec/specs/` is empty; no existing requirements change._

## Impact

- **Code**:
  - `static/templates/admin.html`, `admin-header.html`, `admin-footer.html` — restructure the shell into a 4-tab bottom/side nav; remove the in-DOM eager renders of inactive panes.
  - `static/templates/admin-*-panes.html` — split into per-tab fragment templates (one template file per tab) serving both the eager (active tab) and lazy (`/api/html/admin/{tab}`) paths.
  - `static/templates/admin-modals.html` — partition modals by tab; ship with their owning tab, not in a global modal block.
  - `main.go` — register `GET /api/html/admin/{tab}` routes (`adminAuth` middleware), set long-lived `Cache-Control` on `/static/**` immutable 1y and `/media/**` 1d revalidate (carried from `fix-admin-ui-perf`), `no-store` on the new tab endpoints and on `/admin.html`.
  - `internal/web/handlers/` (or equivalent admin handler) — new handlers per tab (`TabRace`, `TabSeason`, `TabDrivers`, `TabConfig`) that execute the existing data queries scoped to that tab and render the corresponding fragment template.
  - `static/sw.js` — audit/refresh precache manifest for vendored assets and the dropped CDN URLs.
  - esbuild config — bundle Bootstrap + FontAwesome CSS; content-hash the bundle; emit the vendor files.
  - `package.json` — add `bootstrap` and `@fortawesome/fontawesome-free` as deps the build step vendors.
- **APIs**: New `GET /api/html/admin/{tab}` endpoints (authenticated, HTML fragment responses). No existing API path/method changed. Long-lived `Cache-Control` added to `/static/**` and `/media/**`; no-store on `/admin.html`.
- **Dependencies**: Adds runtime `bootstrap` and `@fortawesome/fontawesome-free` npm deps (already loaded via CDN today; new is the self-hosting). Drops runtime CDN dependency (improves offline resilience and removes render-blocking third-party origin).
- **Systems**: Service worker precache must reflect the new vendored bundle paths; stale CDN entries removed.
- **Breaking**: Internal admin HTML route shape changes only; admin HTML is consumed by the admin UI itself, not a public API. **No public API breaking change**. The top-level `/admin.html` route still renders the shell; subtab content now arrives via the new `/api/html/admin/{tab}` family.
- **Relationship to `fix-admin-ui-perf`**: Supersedes it. That change's CSS-preload (D3) is obviated by self-hosting; its `.tab-pane` 150ms cap (D4) is replaced by the `prefers-reduced-motion`-aware handling here; its Cache-Control tasks (D2) are absorbed as one task; its lazy-mount non-goal becomes the core mechanism. Recommend archiving `fix-admin-ui-perf` upon implementation of this change.