## Context

The service worker (`static/sw.js`) precaches a hardcoded list of URLs on install. Since the `redesign-admin-ui` change (commit 7852475) self-hosted Bootstrap + FontAwesome via esbuild content-hashed bundles, the precache list must track those hashed filenames. `build.mjs` emits `static/vendor/manifest.json` with the current hashed paths (`bootstrapCss`, `bootstrapJs`, `fontawesomeCss`, `adminNavCss`), but `sw.js` is hand-edited and has drifted: it references `admin-nav.a01ce8d3267f.css` while the actual file on disk (and in the manifest) is `admin-nav.ca3c352da7b9.css`. On install, `cache.addAll(PRECACHE_URLS)` hits a 404 for the stale path → the entire precache rejects → on refresh the SW has no cached shell → bare-bones UI with no CSS. This is the reported regression.

The SW is registered in `static/templates/index.html:149-150` (`navigator.serviceWorker.register('/sw.js')`) and served by `main.go:611-613`. The `activate` handler (sw.js:36-41) already evicts caches whose key ≠ `CACHE`, so a cache-version bump is the standard invalidation mechanism. CI (`.github/workflows/ci.yaml`) runs `npm run build:frontend` (which runs `build.mjs`) but does not assert that `sw.js`'s precache list matches the build output — so the drift shipped undetected.

## Goals / Non-Goals

**Goals:**
- Make `sw.js`'s vendor precache entries impossible to drift from `build.mjs` output by generating them from `manifest.json` at build time.
- Add a CI assertion that every precache path resolves to a file on disk, so any future drift (manual edit, manifest schema change, missing build step) fails the pipeline.
- Invalidate the broken `heat-cache-v3` cache on deployed clients via a version bump to `heat-cache-v4`.

**Non-Goals:**
- Rewriting the SW architecture (stale-while-revalidate, runtime caching strategy, etc.). The current cache-first-with-network-fallback strategy stays.
- Precaching the per-tab admin fragments (`/api/html/admin/*`) — those are `no-store` by design.
- Tightening the CSP or self-hosting htmx (admin-footer.html:2 still loads from unpkg). Separate changes.
- Archiving `redesign-admin-ui` or `fix-admin-ui-perf` — those have unchecked manual-verify tasks; archive after this change lands (it satisfies task 7.3 of `redesign-admin-ui`).

## Decisions

### D1 — Generate `sw.js` from a template at build time, injecting `PRECACHE_URLS` from `manifest.json`
**Rationale**: `build.mjs` already produces `manifest.json` as the single source of truth for vendor asset paths. Reading that manifest and injecting its paths into `sw.js` makes drift structurally impossible — a human cannot typo a hash into `sw.js` because the array is generated. The non-vendor precache entries (`/`, `/static/style.css`, `/static/favicon.svg`, `/static/manifest.json`) and the SW logic (install/fetch/activate handlers) stay in a hand-maintained `static/sw.template.js` with a `__PRECACHE_URLS__` token. Only the array is generated.
**Alternatives considered**:
- Keep `sw.js` hand-edited, add only the CI guard. Rejected: the guard catches drift but a human still has to update `sw.js` on every vendor bump — easy to forget, and the failure mode (broken precache) is severe.
- Generate the entire `sw.js` from `build.mjs` (no template). Rejected: the SW logic is stable and hand-maintained; generating it from JS risks reformatting noise on every build and makes the SW harder to read in the repo.
- Use Workbox for precache manifest generation. Rejected: adds a runtime dependency for a 42-line SW; overkill.

### D2 — Inject the cache version from `build.mjs` too, via a `__CACHE_VERSION__` token
**Rationale**: Bumping the cache version is the standard SW-invalidation step. Generating it from `build.mjs` means a vendor-content change (which changes a hash → changes the precache array) can automatically bump the version, so deployed clients always drop the old cache when assets change. For this change, bump `v3` → `v4` manually to invalidate the currently-broken deployed caches; future bumps can be automated by hashing the precache array.
**Alternatives considered**: Keep the version hand-edited in the template. Rejected: easy to forget on a vendor bump, leaving clients with a stale cache that points at deleted hashed files — exactly this bug.

### D3 — CI guard via `scripts/verify-sw-precache.mjs`, run after `Build TypeScript`
**Rationale**: The build produces both `manifest.json` and `sw.js`; the guard parses `sw.js`'s `PRECACHE_URLS` (regex on the generated array), resolves every `/static/vendor/*` path against the filesystem, and exits non-zero if any file is missing or if any manifest path is absent from `sw.js`. Catches: (a) someone editing `sw.js` by hand and reintroducing drift, (b) a build step that writes the manifest but fails to regenerate `sw.js`, (c) a manifest schema change that drops a key. Runs as `npm run check:sw` and as a CI step in `ci.yaml` after `Build TypeScript` (so the vendor files exist on the runner).
**Alternatives considered**:
- Assert in a Go test. Rejected: the build is Node-driven; the guard belongs in the Node toolchain next to `build.mjs`.
- Assert via a Playwright test that checks SW install. Rejected: slower, flakier, and checks the wrong layer (browser SW runtime vs. build-output consistency).

### D4 — Check in the generated `static/sw.js` (not gitignore it)
**Rationale**: The repo is buildable for Go-only consumers without a Node step (the `go build` path serves the checked-in `static/`). Gitignoring `sw.js` would break that. The CI guard catches drift on PRs; the generated file is deterministic given the same manifest, so check-in noise is minimal (only changes when a vendor hash changes, which is exactly when it should).
**Alternatives considered**: Gitignore `sw.js` and generate at deploy. Rejected: breaks Go-only builds; diverges from how `static/js/*.js` (also build outputs) are currently checked in.

## Risks / Trade-offs

- **[Generated `sw.js` reformatting noise on every build]** → Mitigation: `build.mjs` writes the array with stable 2-space indentation and a sorted order (non-vendor first, then manifest keys in fixed order); the output is deterministic. Diff noise only when the precache set actually changes.
- **[CI guard passes but SW still broken at runtime]** → Mitigation: the guard checks filesystem existence, not HTTP reachability; the existing `r.GET("/sw.js")` and `r.Static("/static")` routes are unchanged. A runtime smoke test is task 7.3 of `redesign-admin-ui` (manual); this change's tasks include a build-time unit assertion that covers the contract.
- **[Cache-version bump forces a full re-download for all deployed clients]** → Mitigation: this is intended — the current `v3` cache is poisoned. Future automated version bumps (D2) will only fire when the precache set actually changes, so clients re-download only when assets genuinely changed.
- **[Someone edits `static/sw.js` directly instead of the template]** → Mitigation: the CI guard catches mismatches between `sw.js` and `manifest.json`; add a header comment in `sw.js` ("DO NOT EDIT — generated from sw.template.js by build.mjs") pointing contributors to the template.
