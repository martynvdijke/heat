## 1. SW template + build generation

- [ ] 1.1 Create `static/sw.template.js` from the current `static/sw.js` body, replacing the hardcoded `PRECACHE_URLS` array with a `__PRECACHE_URLS__` token and the `CACHE` string with a `__CACHE_VERSION__` token. Keep the install/fetch/activate handlers unchanged.
- [ ] 1.2 Add a generated-file header comment to the template top: `// DO NOT EDIT — generated from static/sw.template.js by build.mjs. Edit the template instead.`
- [ ] 1.3 Extend `build.mjs`: after `buildVendor()` writes `manifest.json`, read the manifest, build the `PRECACHE_URLS` array = `['/', '/static/style.css', '/static/favicon.svg', '/static/manifest.json']` + `[manifest.bootstrapCss, manifest.bootstrapJs, manifest.fontawesomeCss, manifest.adminNavCss]`, read `static/sw.template.js`, replace `__PRECACHE_URLS__` with a JSON-stringified array and `__CACHE_VERSION__` with `'heat-cache-v4'`, write `static/sw.js`.
- [ ] 1.4 Run `npm run build:frontend` and confirm `static/sw.js` now references `admin-nav.ca3c352da7b9.css` (the real hash) and `CACHE = "heat-cache-v4"`.
- [ ] 1.5 Confirm the generated `sw.js` is byte-stable across repeated builds with no manifest change (deterministic output).

## 2. CI guard

- [ ] 2.1 Create `scripts/verify-sw-precache.mjs`: parse `static/sw.js` with a regex extracting the `PRECACHE_URLS` array; parse `static/vendor/manifest.json`; assert every manifest path (`bootstrapCss`, `bootstrapJs`, `fontawesomeCss`, `adminNavCss`) is present in the array; assert every `/static/vendor/*` path in the array resolves to a file on disk; exit non-zero with a clear message on any mismatch.
- [ ] 2.2 Add `"check:sw": "node scripts/verify-sw-precache.mjs"` to `package.json` scripts.
- [ ] 2.3 Add a `check:sw` task to `Taskfile.yml` mirroring the npm script.
- [ ] 2.4 Add a `Run SW precache check` step to `.github/workflows/ci.yaml` after `Build TypeScript`, running `npm run check:sw`.
- [ ] 2.5 Run `npm run check:sw` locally; confirm it passes on the regenerated `sw.js` and fails when a vendor path in `sw.js` is deliberately corrupted.

## 3. Cache invalidation

- [ ] 3.1 Confirm the generated `sw.js` uses `heat-cache-v4` (not `v3`).
- [ ] 3.2 Confirm the existing `activate` handler (sw.js:36-41) evicts `heat-cache-v3` on next activation (it deletes any cache key ≠ `CACHE`).

## 4. Tests

- [ ] 4.1 Add a Node unit test (or extend `verify-sw-precache.mjs` with a `--test` mode) that runs `build.mjs`, then asserts every `manifest.json` path appears in `sw.js`'s `PRECACHE_URLS`.
- [ ] 4.2 Add a negative test: corrupt `sw.js`'s admin-nav path, run the guard, assert non-zero exit. (Can be a separate script or a `--self-test` flag.)
- [ ] 4.3 `go test ./...` green (no Go changes expected, but confirm no regression).

## 5. Manual verification

- [ ] 5.1 Hard-reload `/admin.html` in a browser; confirm CSS loads on first paint.
- [ ] 5.2 Refresh `/admin.html`; confirm CSS still present (SW precache no longer rejects).
- [ ] 5.3 In DevTools → Application → Service Workers, confirm the SW registers, installs, and precaches with no 404s for vendor assets.
- [ ] 5.4 In DevTools → Application → Cache Storage, confirm `heat-cache-v4` contains all `PRECACHE_URLS` entries and `heat-cache-v3` is gone.
- [ ] 5.5 Satisfies `redesign-admin-ui` task 7.3 (SW smoke test). Note this in the change's design.md or a closing comment so 7.3 can be checked there.

## 6. Cleanup

- [ ] 6.1 `task pre-push` green (gofmt, go test, govulncheck, tsc, go build).
- [ ] 6.2 Commit with `fix: generate sw.js precache from vendor manifest and add CI guard`.
