## Context

The admin page is a single Go-template-rendered HTML document (`/admin.html` route, main.go:533) containing all 5 management categories (Race/Results/Content/Settings/System), every subtab, and every modal in the DOM at once — ~1181 lines. Static assets (`/static`, `/media`, `/sw.js`) are served by gin's `r.Static`/`r.StaticFile` at main.go:525-527 with no `Cache-Control` headers. The admin `<head>` (templates/admin-header.html) loads Bootstrap 5.3.0 CSS and FontAwesome 6.7.2 CSS from CDN as render-blocking `<link rel="stylesheet">` tags. A stale, unserved duplicate `static/admin.html` (1167 lines, FontAwesome 7.0.1, different element IDs) co-exists and risks confusion or service-worker precache conflicts. Users observe a ~10s animation followed by slow paint when switching tabs — a regression from v1.30.

## Goals / Non-Goals

**Goals:**
- Restore tab-switch latency to ≤200ms (v1.30 parity).
- Eliminate render-blocking CSS from the admin first paint.
- Make static/media assets cacheable on repeat navigations.
- Remove the dead duplicate `static/admin.html` from the repo.
- Keep CDN URLs, asset versions, route paths, and markup semantics unchanged — pure perf fix.

**Non-Goals:**
- Self-hosting Bootstrap / FontAwesome (deferred; preload pattern achieves the render-block win without a download-migration).
- Splitting the admin into multiple routes or lazy-mounting subtab panes via htmx (deferred; revisit only if the four goals above don't restore v1.30 speed).
- Backend query optimization (no evidence of slow handlers; symptom is pure front-end).
- I18n or middleware changes.
- Service worker architecture rewrite — only the precache list is audited for the deleted file.

## Decisions

### D1 — Delete `static/admin.html` instead of keeping it as a backup
**Rationale**: It is not referenced by any Go route (verified the only `/admin.html` handler serves the parsed template tree, not this file). Keeping it invites future bugs — a service worker or contributor could pick it up.不需要 of死代码.
**Alternative considered**: Move to an `archive/` folder. Rejected: increases repo size forever for no ongoing value; git history preserves it.

### D2 — Set `Cache-Control` per route at the gin handler level, not via middleware
**Rationale**: Different directives per route family (`/static` immutable 1y because esbuild content-hashes bundles; `/media` 1d because media may change; `/sw.js` no-cache because the SW must always revalidate). A per-route wrapper around `r.Static` is clearer than a path-matching middleware and avoids accidentally caching the admin HTML.
**Alternative considered**: A single `CacheMiddleware` keyed on URL prefix. Rejected: more indirection, harder to audit, easier to misconfigure.

### D3 — Preload+async-load the CSS via `rel="preload" ... onload="this.rel='stylesheet'"`
**Rationale**: Standard progressive-enhancement pattern; preserves the same CDN URLs and the same final CSSOM. Avoids FOUC on the admin shell because the visible-on-load nav skeleton has minimal styling needs, and Bootstrap's `display:none` on `.tab-pane` (the inactive panes) is what hides them initially — that is set inline by the template's `active` class, not by the CSS file.
**Alternative considered**: Inline critical CSS for the nav shell and async-load the rest. Rejected: adds maintenance burden and a build step; preload pattern is simpler and sufficient.

### D4 — Cap `.tab-pane` transition duration to 150ms via a custom override file
**Rationale**: Bootstrap's default `.fade` transition is 150ms — if the observed 10s animation is a custom rule (e.g. `transition: all 1s` or a keyframe), capping `.tab-pane` to 150ms restores Material-guidance-compliant timing. Override loaded after Bootstrap CSS so it wins on specificity.
**Alternative considered**: Remove the `fade` class from pane templates. Rejected: makes the cap invisible at the CSS level and risks regressions if Bootstrap's default ever changes; an explicit override is self-documenting.
**Note**: The exact offending rule must be located before the override is written (Task T4 in tasks.md). If no custom rule exists and Bootstrap's default `.fade` is the actual cause, the override still works — `.tab-pane.fade { transition: opacity 150ms ease-in-out; }`.

### D5 — Audit `sw.js` precache after deleting `static/admin.html`
**Rationale**: If the service worker precaches paths matching `admin.html`, deleting the file leaves a stale precache entry that could serve a 404 or fail install. Audit before/after.
**Alternative considered**: Rewrite `sw.js` to drop precache entirely. Rejected: out of scope; precache is fine if its list matches served assets.

## Risks / Trade-offs

- **[FOUC during initial CSS load]** → Mitigation: preload + the template's inline `active`/`d-none` classes hide inactive panes before Bootstrap loads; the nav bar itself is plain HTML that renders without Bootstrap's grid. Acceptable; if visible flicker is reported, add a tiny inline critical CSS for the nav shell in a follow-up.
- **[Service worker precache now references a missing file]** → Mitigation: grep `sw.js` for `admin.html` and any precache manifest before deleting; remove stale entries in the same commit.
- **[Caching immutable for 1y causes stale JS if esbuild ever emits same hash for different content]** → Mitigation: esbuild's content hash is collision-safe in practice; if a misconfiguration produces a stale bundle, bump the cache-busting query param on the `<script src>` in `admin-header.html` or `admin-footer.html`.
- **[Custom CSS override loads before Bootstrap on cached load]** → Mitigation: ensure the override `<link>` is emitted after Bootstrap's preload/link in `admin-header.html` (or use `body`-level class to win specificity).
- **[10s animation turns out to be a JS-driven animation, not CSS]** → Mitigation: tasks.md includes a read-only investigation step before the cap; if JS is the cause, the design adapts (likely a removed `requestAnimationFrame` loop or a shorter duration arg) without changing the spec.