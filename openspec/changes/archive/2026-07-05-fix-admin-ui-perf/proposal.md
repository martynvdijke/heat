## Why

The admin UI regressed badly versus v1.30: clicking a top or side nav tab triggers a ~10s animation followed by slow content paint, making the panel feel broken. Root causes are front-end only — render-blocking external CSS (Bootstrap + FontAwesome) in `<head>`, an uncapped/oversized CSS transition on large swapped panes, a monolithic 1181-line DOM that holds all 5 categories × all subtabs × all modals at once, no `Cache-Control` on `/static` `/media` `/sw.js`, and a dead duplicate `static/admin.html` that risks service-worker precache conflicts.

## What Changes

- Delete the dead duplicate `static/admin.html` (not served by any Go route; purely a maintenance hazard).
- Add long-lived `Cache-Control` headers to the static asset routes in `main.go` (`/static/**` immutable 1y, `/media/**` 1d revalidate, `/sw.js` no-cache).
- De-render-block Bootstrap and FontAwesome CSS in `admin-header.html` using `<link rel="preload" ... onload="this.rel='stylesheet'">` + `<noscript>` fallback; no CDN swap, no self-host.
- Identify and cap/disable the long-running CSS transition on admin tab panes (suspect Bootstrap `.fade` on `.tab-pane` or a custom `transition: all` rule) so tab swaps complete in ≤200ms.
- Optional / deferred: lazy-mount heavy subtab panes via htmx `hx-get` on first activation instead of rendering the entire admin tree upfront. **Out of scope for this change** unless the items above fail to restore v1.30 speed.

## Capabilities

### New Capabilities
- `admin-asset-cache`: Long-lived `Cache-Control` headers for `/static`, `/media`, `/sw.js` routes so repeat navigations use cached assets.
- `admin-non-blocking-css`: Bootstrap and FontAwesome CSS load without blocking first paint of the admin shell.

### Modified Capabilities
_None — `openspec/specs/` is empty; no existing requirements change._

## Impact

- **Code**: `main.go` (static route handlers ~lines 525-527), `templates/admin-header.html` (CSS link tags), one custom CSS override file or inline block for `.tab-pane` transition, deletion of `static/admin.html`.
- **APIs**: HTTP response headers on `/static/*`, `/media/*`, `/sw.js` only — no path or method changes.
- **Dependencies**: No new go modules, no npm packages. Existing CDN URLs unchanged.
- **Systems**: Service worker (`sw.js`) must continue to precache only served assets; precache manifest audited after `static/admin.html` deletion.
- **Breaking**: None. Pure performance/UX fix; no public API change.