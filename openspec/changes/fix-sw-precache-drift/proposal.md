## Why

The admin UI (and every service-worker-cached page) renders bare-bones without CSS after a refresh. Root cause: `static/sw.js` hardcodes a stale content-hashed vendor path (`admin-nav.a01ce8d3267f.css`) that no longer exists on disk, so `cache.addAll(PRECACHE_URLS)` rejects on install and the precache stays empty. `build.mjs` regenerates content-hashed vendor files and writes `manifest.json`, but never updates `sw.js` — the precache list is hand-edited and drifts on every vendor rebuild. This is the unchecked task 7.3 from `redesign-admin-ui`.

## What Changes

- **Generate `sw.js` precache from `manifest.json` at build time.** `build.mjs` templates `static/sw.js` from a hand-maintained `static/sw.template.js`, injecting `PRECACHE_URLS` = static non-vendor entries (`/`, `/static/style.css`, `/static/favicon.svg`, `/static/manifest.json`) ∪ all paths from `manifest.json` (`bootstrapCss`, `bootstrapJs`, `fontawesomeCss`, `adminNavCss`). Single source of truth = the vendor manifest. Hand-editing vendor paths in `sw.js` becomes impossible.
- **Add a CI guard.** New `scripts/verify-sw-precache.mjs` parses `static/sw.js`'s `PRECACHE_URLS`, asserts every `/static/vendor/*` path resolves to a file on disk, and exits non-zero on any mismatch. Wired as `npm run check:sw` and added as a CI step in `.github/workflows/ci.yaml` after `Build TypeScript`. PRs that drift fail CI.
- **Bump the SW cache version** from `heat-cache-v3` to `heat-cache-v4` so deployed clients drop the poisoned cache on next load.
- **Fix the immediate stale hash** by regenerating `sw.js` from the template (corrects `admin-nav.a01ce8d3267f.css` → `admin-nav.ca3c352da7b9.css`).

## Capabilities

### New Capabilities
- `sw-precache-from-manifest`: The service worker precache list is generated from the build's vendor manifest at build time, and CI asserts every precache path resolves to a file on disk. Eliminates hand-edited drift between `build.mjs` output and `sw.js`.

### Modified Capabilities
_None — `openspec/specs/` is empty; no existing requirements change. The `admin-vendored-assets` spec stub inside `redesign-admin-ui` describes vendor self-hosting but does not cover SW precache generation; this change adds that contract as a new capability._

## Impact

- **Code**:
  - `build.mjs` — read `manifest.json` after vendor build, inject `PRECACHE_URLS` into `static/sw.js` from `static/sw.template.js`.
  - NEW `static/sw.template.js` — current `sw.js` body with a `__PRECACHE_URLS__` token and a `__CACHE_VERSION__` token.
  - NEW `scripts/verify-sw-precache.mjs` — read-only assertion script.
  - `static/sw.js` — regenerated output (checked in, not gitignored, so the repo is buildable without a node step for Go-only consumers); bumped to `heat-cache-v4` with the corrected admin-nav hash.
  - `package.json` — add `check:sw` script.
  - `Taskfile.yml` — add `check:sw` task.
  - `.github/workflows/ci.yaml` — add `Run SW precache check` step after `Build TypeScript`.
- **APIs**: None. `sw.js` is served by the existing `r.GET("/sw.js", ...)` handler (main.go:611); its content changes, its route does not.
- **Dependencies**: No new runtime deps. Uses Node built-ins (`fs`, `path`, `crypto`) already used by `build.mjs`.
- **Systems**: Deployed clients with the broken `heat-cache-v3` cache will activate the new `heat-cache-v4` SW, delete the old cache (existing `activate` handler already evicts non-matching cache keys), and precache the correct vendor set.
- **Breaking**: None. SW update is transparent to end users; the cache-version bump is the standard SW-invalidation mechanism.
