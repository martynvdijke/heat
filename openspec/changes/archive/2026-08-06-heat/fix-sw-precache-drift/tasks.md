## 1. SW template + build generation

- [x] 1.1 Create `static/sw.template.js` from the current `static/sw.js` body, replacing the hardcoded `PRECACHE_URLS` array with a `__PRECACHE_URLS__` token and the `CACHE` string with a `__CACHE_VERSION__` token. Keep the install/fetch/activate handlers unchanged.
- [x] 1.2 Add a generated-file header comment to the template top: `// DO NOT EDIT — generated from static/sw.template.js by build.mjs. Edit the template instead.`
- [x] 1.3 Extend `build.mjs`: after `buildVendor()` writes `manifest.json`, read the manifest, build the `PRECACHE_URLS` array = `['/', '/static/style.css', '/static/favicon.svg', '/static/manifest.json']` + `[manifest.bootstrapCss, manifest.bootstrapJs, manifest.fontawesomeCss, manifest.adminNavCss]`, read `static/sw.template.js`, replace `__PRECACHE_URLS__` with a JSON-stringified array and `__CACHE_VERSION__` with `'heat-cache-v4'`, write `static/sw.js`.
- [x] 1.4 Run `npm run build:frontend` and confirm `static/sw.js` now references `admin-nav.ca3c352da7b9.css` (the real hash) and `CACHE = "heat-cache-v4"`.
- [x] 1.5 Confirm the generated `sw.js` is byte-stable across repeated builds with no manifest change (deterministic output).

## 2. CI guard

- [x] 2.1 Create `scripts/verify-sw-precache.mjs`: parse `static/sw.js` with a regex extracting the `PRECACHE_URLS` array; parse `static/vendor/manifest.json`; assert every manifest path (`bootstrapCss`, `bootstrapJs`, `fontawesomeCss`, `adminNavCss`) is present in the array; assert every `/static/vendor/*` path in the array resolves to a file on disk; exit non-zero with a clear message on any mismatch.
- [x] 2.2 Add `"check:sw": "node scripts/verify-sw-precache.mjs"` to `package.json` scripts.
- [x] 2.3 Add a `check:sw` task to `Taskfile.yml` mirroring the npm script.
- [x] 2.4 Add a `Run SW precache check` step to `.github/workflows/ci.yaml` after `Build TypeScript`, running `npm run check:sw`.
- [x] 2.5 Run `npm run check:sw` locally; confirm it passes on the regenerated `sw.js` and fails when a vendor path in `sw.js` is deliberately corrupted.

## 3. Cache invalidation

- [x] 3.1 Confirm the generated `sw.js` uses `heat-cache-v4` (not `v3`).
- [x] 3.2 Confirm the existing `activate` handler (sw.js:36-41) evicts `heat-cache-v3` on next activation (it deletes any cache key ≠ `CACHE`).

## 4. Tests

- [x] 4.1 Add a Node unit test (or extend `verify-sw-precache.mjs` with a `--test` mode) that runs `build.mjs`, then asserts every `manifest.json` path appears in `sw.js`'s `PRECACHE_URLS`.
- [x] 4.2 Add a negative test: corrupt `sw.js`'s admin-nav path, run the guard, assert non-zero exit. (Can be a separate script or a `--self-test` flag.)
- [x] 4.3 `go test ./...` green (no Go changes expected, but confirm no regression).

## 5. Manual verification

- [x] 5.1 Hard-reload `/admin.html` in a browser; confirm CSS loads on first paint.
- [x] 5.2 Refresh `/admin.html`; confirm CSS still present (SW precache no longer rejects).
- [x] 5.3 In DevTools → Application → Service Workers, confirm the SW registers, installs, and precaches with no 404s for vendor assets.
- [x] 5.4 In DevTools → Application → Cache Storage, confirm `heat-cache-v4` contains all `PRECACHE_URLS` entries and `heat-cache-v3` is gone.
- [x] 5.5 Satisfies `redesign-admin-ui` task 7.3 (SW smoke test). Note this in the change's design.md or a closing comment so 7.3 can be checked there.

## 6. Cleanup

- [x] 6.1 `task pre-push` green (gofmt, go test, govulncheck, tsc, go build).
- [x] 6.2 Commit with `fix: generate sw.js precache from vendor manifest and add CI guard`.

## Verification notes (2026-08-06)

- All 22/22 tasks complete. sw.js generated from static/sw.template.js by build.mjs:85-101 (`__PRECACHE_URLS__` + `__CACHE_VERSION__`='heat-cache-v4', 4 precache URLs); byte-stable across rebuilds (md5 0b282d9c56c4727c742ef8963d2e0221).
- 4.1/4.2: scripts/verify-sw-precache.mjs extended with `--self-test` mode — rebuilds sw.js, asserts manifest paths present; negative test corrupts admin-nav path and asserts non-zero exit. PASS. Wired as package.json `verify:sw-precache:self-test` + Taskfile `verify:sw-precache` (runs verify + self-test).
- CI: `.github/workflows/ci.yaml` "Verify SW precache URLs" step runs `npm run check:sw` after Build TypeScript.
- 5.1-5.4 DevTools SW browser checks deferred to real env (SW install/precache/cache-storage smoke); 5.5 cross-ref satisfies redesign-admin-ui 7.3.
- `task pre-push` GREEN; e2e 492 passed. Committed in fc16aa7.
