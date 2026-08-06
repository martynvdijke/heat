## Context

The admin UI today (`main.go:543` handler for `/admin.html`, `initAdminTemplate` at main.go:138) renders one Go template tree containing 9 fragment files (~1284 lines of HTML) into a single HTML response. Bootstrap 5.3.0 CSS and FontAwesome 6.7.2 CSS are pulled from CDN as render-blocking `<link rel="stylesheet">`. Switching a tab swaps a `.tab-pane` whose `display` toggles, but the entire sub-DOM is already in the document. On phones the top `.nav-tabs` wraps/scrolls; on all devices the upfront rendering cost is felt as the ~10s animation/slow-paint regression reported in `fix-admin-ui-perf`. Static assets are already wrapped by a `cacheControl(...)` middleware on the gin `r.Group("/static")` (main.go:532) — that mechanism is available for reuse.

The Go side already has a pattern for returning HTML fragments under admin auth: `admin.GET/POST/PUT/DELETE("/html/<entity>/:id", h.Htmx...)` handlers (main.go:380-406). The redesign extends that pattern with whole-tab fragment endpoints rather than per-row edit snippets.

Stakeholder intent: a touch-friendly admin that paints fast on a cold load, with workflow-logical grouping (Race Day / Season / Drivers / Config).

## Goals / Non-Goals

**Goals:**
- Touch-first bottom bar with 4 workflow tabs; ≥44px targets; safe-area aware; lifts to a side rail on ≥768px.
- Lazy-mount inactive panes via htmx so the initial admin DOM contains only the active tab's HTML+modals.
- Eliminate render-blocking CDN round-trips by self-hosting Bootstrap + FontAwesome, bundled & hashed by esbuild.
- New `GET /api/html/admin/{tab}` endpoint family authenticated by the existing admin middleware, returning one tab's HTML fragment.
- Client-side first-load cache of mounted panes so re-activating a tab is instant (no second request).
- Total tab-switch latency ≤200ms warm, ≤1s cold (over the wire).
- `prefers-reduced-motion` short-circuit for fade and bar animations.
- Carry over the long-lived `Cache-Control` for `/static/**` and `/media/**` and `no-store` for `/admin.html` and the new tab endpoints.

**Non-Goals:**
- Oauth-configured CDN alternate URLs — the CDN goes away entirely.
- Backend query optimization; existing handlers' query plans are assumed adequate (see `fix-admin-ui-perf` design D4 "no evidence of slow handlers").
- Splitting the admin into multiple top-level routes (`/admin.html` stays the shell; everything below becomes fragment endpoints).
- Redesigning modals' UX (modal structure is preserved; their placement in the DOM moves with their owning tab).
- Migrating to a SPA framework (Vue/React). Stays Go templates + htmx.
- Re-skinning the live board, controller, pit board, or any other public page — admin only.

## Decisions

### D1 — Bottom bar responsive-lifts to a side rail via one shared DOM, not two navs
**Rationale**: A single `<nav id="admin-nav">` rendered in the admin shell is reflowed by CSS media queries — `position: fixed; bottom: 0` below 768px, `position: sticky; left: 0; width: 220px; height: 100vh` at ≥768px. The same `<button>` elements, ARIA semantics, and onclick handlers serve both contexts. Halves maintenance vs. two synced navs and matches what Bootstrap's own responsive nav concepts aim at.
**Alternatives considered**:
- Two nav DOMs toggled by class. Rejected: duplicated IDs/tests.
- Bottom bar everywhere, including desktop. Rejected: wide-screen thumb distance is irrelevant on desktop; a side rail reads as "admin console" and uses vertical real estate well.
- Keep top `.nav-tabs` for desktop, bottom bar for mobile. Rejected: two navs to keep in sync; cognitive overhead of finding different things in different places.

### D2 — Tab count is 4: Race Day, Season, Drivers, Config
**Rationale**: The industry bottom-bar sweet spot is 3–5; iOS HIG caps at 5. 4 leaves comfortable thumb spacing. The 4-group workflow taxonomy (operate / curate championship / curate people / configure) covers all 17 today's subtabs without orphaning anything. Crucially, **Settings (7 subtabs)** merged with **System (Logs)** into one Config tab keeps the bar at 4 and respects that both are "operator-admin" cover-of-detail; the 7-vs-Logs split can live as a sub-nav inside Config.
**Alternatives considered**:
- 5 tabs preserving Settings vs System. Rejected: bottom-bar tighter; the Settings/System distinction is internal-only and a Config sub-nav surfaces it for the operators who care.
- 3 tabs merging Drivers into Season. Rejected: Racers/Teams/Quotes are distinct enough workflow-wise and the Drivers tab never has to be loaded by users only running race-day ops.
- Tab = each current category (Race, Results, Content, Settings, System). Rejected: just relocates the same tile pain to the bottom.

### D3 — Lazy-mount via htmx `hx-get` on tab activation; client caches mounted panes
**Rationale**: Htmx is already loaded (admin-footer.html:2) and used for row-level edits. Extending it to whole-tab fragments is one new endpoint per tab and one `<div id="admin-tab-container" hx-get="/api/html/admin/race-day" hx-trigger="load">` on initial load. Tab buttons issue `hx-get="/api/html/admin/<tab>"` with `hx-target="#admin-tab-container"` and `hx-swap="innerHTML"`. Mounting replaces the container's children; a `data-tab-mounted="<id>"` attribute on the container lets the click handler short-circuit if already mounted (warm cache, ≤200ms).
**Alternatives considered**:
- Mount every pane and CSS-hide inactive ones (current behaviour). Rejected: this is the source of the 10s paint regression.
- Use `<details>`/`<dialog>` with hidden inactive content. Rejected: same upfront DOM cost.
- Lazy-mount via vanilla JS `fetch`. Rejected: htmx already wired; using two systems is fragmentation.
- Server keeps every tab eagerly rendered but only transmits the active one with a `data-tab-fragment` header. Rejected: still pays the server-side render cost for inactive tabs on every nav.

### D4 — Self-host Bootstrap + FontAwesome, bundle+hash with esbuild
**Rationale**: The redesign already touches `admin-header.html`; promoting the deferred non-goal of `fix-admin-ui-perf` here means no more render-blocking CDN round-trip, no FOUC waiting on a third party, and an immutable cacheable bundle (matches the existing `Cache-Control: public, max-age=31536000, immutable` on `/static/**`). Vendor files under `/static/vendor/`, import via the esbuild entry, emit one `admin.<hash>.css` and the existing `admin.<hash>.js` pattern.
**Alternatives considered**:
- Keep CDN, only fix the preload pattern. Rejected: that's `fix-admin-ui-perf`'s scope; we're going further intentionally.
- Self-host but ship the unminified vendor files as-is. Rejected: loses bundling/tree-shaking and the immutable hash cache key.
- Adopt a lighter CSS framework (e.g., Pico). Rejected: Bootstrap components are pervasive in existing templates; rip-and-replace is out of scope.

### D5 — Pane fade capped; `prefers-reduced-motion` short-circuits to no animation
**Rationale**: A 150ms opacity fade on tab swaps gives feedback without feeling slow. `@media (prefers-reduced-motion: reduce)` removes the fade entirely (instant swap) for users who opt out. Cap lives in the new bundled admin CSS.
**Alternatives considered**: Remove the fade class entirely for everyone. Rejected: subtle motion feedback improves perceived performance and discoverability of the switch.

### D6 — Modal placement follows their owning tab
**Rationale**: Today's `admin-modals.html` is a 244-line block rendered for all tabs. Modals that are only used by one tab (e.g., `#racerModal`, `#quoteModal`, `#teamModal`, `#seasonsModal`) ship with their owning tab's fragment. Modals used by multiple tabs are duplicated per tab (cheap — modals are small and only one tab is mounted at a time, so duplication is not a memory/perf cost). Avoids a global modal-registry layer at the shell.
**Alternatives considered**: Keep one global modal block at the shell. Rejected: still ships every modal upfront for the active tab; defeats part of the lazy-mount goal.

### D7 — `GET /api/html/admin/{tab}` returns fragment HTML; `no-store`; admin auth; CSRF exempt (GET)
**Rationale**: GET endpoints need no CSRF token (existing GET handlers in the admin group DO run through `CSRFMiddleware` but allow GET). The `no-store` directive is correct because tab content reflects live race state and must always revalidate. Use the same `middleware.AuthMiddleware(server)` and `middleware.CSRFMiddleware()` group the existing `/html/*` handlers already use (extend that group).
**Alternatives considered**: `Cache-Control: private, max-age=1`. Rejected: a 1s revalidate risks serving a just-stale qualification grid mid-race. no-store is cheaper to reason about.

### D8 — Reuse existing `cacheControl` middleware for the no-store on /admin.html
**Rationale**: `cacheControl` is already used on the `/static` and `/media` groups (main.go:532-533). Wrap the `/admin.html` handler's group with `cacheControl("no-store")` instead of inline `c.Header(...)` for consistency.

## Risks / Trade-offs

- **[Active-tab initial server render is now on the cold path]** → Mitigation: the Race Day tab (default active) probably has the richest queries; measure its render time in a load-test step in tasks.md. If it exceeds ~300ms server-side, pre-fetch the next tab's fragment on `load` via htmx `hx-trigger="load delay:300ms"` (progressive pre-warm).
- **[Tab switch makes a network request — first activation is slow on bad networks]** → Mitigation: bundle the four tab fragments into the initial response? Rejected (defeats the goal). Instead: show a skeleton indicator via `htmx:beforeRequest`, ship the lane of progress visible to the user; ≤1s SLO covers typical LAN. Document offline behaviour (will show error; admin already requires connectivity for live ops).
- **[esbuild bundle grows with vendor CSS — first paint download bigger]** → Mitigation: split vendor CSS into its own `admin-vendor.<hash>.css` loaded after the small critical admin shell CSS so the shell paints first. Tree-shake unused Bootstrap components in a follow-up if bundle size is reported.
- **[Self-hosted FontAwesome is large]** → Mitigation: ship FontAwesome as a separate bundle with `media="print" onload="this.media='all'"` pattern retained so icons progressively appear without blocking; or migrate to FA's "kit"/subset in a follow-up. Tracked as a task to pick the subset.
- **[Tab ID drift between front-end and back-end]** → Mitigation: a single Go const map `{ "race-day", "season", "drivers", "config" }` shared with the template via server-side data, so templates render the same names the router matches.
- **[Service worker precache stale after vendor swap]** → Mitigation: regenerate `sw.js` precache manifest from the esbuild metafile as a build step; grep `sw.js` for old CDN URLs to confirm none survives.
- **[Users with JS disabled lose all admin functionality]** → Mitigation: the active tab works without htmx (server renders it eagerly); switching tabs requires JS, which is documented in the spec. Acceptable — admin already requires JS for live ops.
- **[Existing 11 htmx row handlers (`/api/html/racers/:id` etc.) still serve fragments expecting parent context]** → Mitigation: confirm they use htmx's `hx-target` external-target pattern (target any element id in the DOM, not the parent template). Tasks include a verification step before the lazy-mount wiring lands.
- **[`fix-admin-ui-perf` change is now obsolete]** → Mitigation: archive `fix-admin-ui-perf` as part of this change's final task; its CSS-preload tasks (D3) are obsolete-by-construction. Its remaining valid task (Cache-Control) is task T1 here.

## Migration Plan

1. Add `bootstrap` + `@fortawesome/fontawesome-free` to `package.json` and the esbuild config; emit `static/vendor/admin-vendor.<hash>.css` + `admin-vendor.<hash>.js`. Build step already exists (`task build`).
2. Add `GET /api/html/admin/{tab}` routes under the existing admin group, with handlers that execute the existing per-tab queries and render the new per-tab fragment templates (split from the current 9-pane files into 4 fragment files).
3. Refactor `admin-header.html` to remove CDN links, reference the hashed vendor bundle, and ship the new shared nav DOM. Refactor `admin.html` to render only the active tab's container plus the nav.
4. Move modals into their owning tab's fragment file.
5. Wire htmx `hx-get` on tab buttons and on initial load. Add `htmx:beforeRequest`/`htmx:afterOnLoad` skeleton indicator and `data-tab-mounted` short-circuit.
6. Add the new bundled admin CSS (≤150ms fade cap, `prefers-reduced-motion` short-circuit, bottom-bar layout, side-rail ≥768px, safe-area insets).
7. Wrap `/admin.html` route with `cacheControl("no-store")`; confirm `/static/**` immutable is automatically applied to vendor bundle.
8. Regenerate `sw.js` precache manifest; remove CDN entries; add vendor bundle entries.
9. Smoke-test: cold load, activate each tab, measure paint; check `prefers-reduced-motion` path; check 768px breakpoint.
10. Archive `fix-admin-ui-perf` change once the Cache-Control task is verified here.

**Rollback**: revert the templates, the esbuild config (CDN links back), and the new routes; existing `admin-*-panes.html` files remain in the tree (split but not deleted) so a revert restores the monolith.

## Open Questions

- Should the Config tab's internal Settings-vs-System split be a second-level bottom bar on phones, a sub-nav at the top of the Config pane, or accordion sections? (Tended toward top sub-nav inside the pane; flagged in tasks as a small UX spike.)
- Should we pre-warm the Season tab on `load` (likely second-most-used) — depends on which tab the live-day users actually need in parallel.
- Does the live board's WebSocket need to push updates into lazy-mounted panes not currently mounted? (If a `/api/html/admin/drivers` fetch should reflect interim racer changes, design holds — current handlers already query fresh on every request. Confirm during implementation.)