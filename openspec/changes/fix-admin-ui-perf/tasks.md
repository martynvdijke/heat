## 1. Dead-code cleanup

- [x] 1.1 Grep `sw.js` (and any precache manifest generator in TS/JS) for `admin.html` references; record findings
- [x] 1.2 Remove any precache entry that resolves to `static/admin.html` from `sw.js`
- [x] 1.3 `git rm static/admin.html`
- [x] 1.4 Verify no Go route references the deleted file (`rg "static/admin.html" --type go`) — updated test in paths_test.go to validate template files instead

## 2. Static asset Cache-Control headers

- [x] 2.1 In `main.go` replace `r.Static("/static", ...)` with wrapped version setting `Cache-Control: public, max-age=31536000, immutable`
- [x] 2.2 Replace `r.Static("/media", ...)` with wrapped version setting `Cache-Control: public, max-age=86400`
- [x] 2.3 Replace `r.StaticFile("/sw.js", ...)` with handler setting `Cache-Control: no-cache`
- [x] 2.4 Confirmed `/admin.html` route (in separate `pages` group) does NOT inherit cache headers
- [x] 2.5 `go vet ./...` and `go test ./...` pass clean

## 3. De-render-block admin CSS

- [x] 3.1 Read and located: Bootstrap at `static/templates/admin-header.html:11` (render-blocking `<link rel="stylesheet">`), FA at line 12 (already non-blocking via `media="print"`)
- [x] 3.2 Converted Bootstrap link to `<link rel="preload" as="style" ... onload="this.rel='stylesheet'">`; FA kept as-is (already uses `media="print" onload="this.media='all'"` pattern)
- [x] 3.3 Added `<noscript>` fallbacks for both Bootstrap and FontAwesome
- [x] 3.4 No rebuild needed — template references `/static/js/admin.js` directly (not hashed); `npm run typecheck` passes
- [ ] 3.5 Manual: Hard-reload `/admin.html` in a browser; confirm no FOUC on the nav shell

## 4. Investigate and cap the long tab transition

- [x] 4.1 Grep done: admin page uses only inline styles (no `static/css/` loaded), no long-duration CSS transition found on `.tab-pane` or `.fade` in admin context
- [x] 4.2 Admin JS (`static/js/admin.js`) is compiled/minified — no source `.addClass('fade')` etc found in the source TS directories
- [x] 4.3 Recorded finding: No 10s CSS rule exists in admin page. "Animation" is likely Bootstrap tab JS + heavy DOM + network fetch time. Safety override still added.
- [x] 4.4 Added `.tab-pane.fade { transition: opacity 150ms ease-in-out; }` to inline `<style>` in `admin-header.html`
- [x] 4.5 Override is inside the inline `<style>` block which has later DOM order than the `<link rel="preload">`, and `.tab-pane.fade` has higher specificity than Bootstrap's `.fade` — guaranteed to win
- [ ] 4.6 Manual: open `/admin.html`, click each top-nav and side-nav tab; verify visible transition completes in ≤200ms
- [ ] 4.7 Manual: DevTools Performance → record a tab click; confirm no Layout/Recalculate Style task exceeds ~100ms for the swap

## 5. Verification

- [x] 5.1 All automated checks pass: `gofmt`, `go test`, `go vet`, `go build`, `npm run typecheck` — all clean
- [ ] 5.2 Manual: DevTools Network — confirm `/static/*` responses carry `Cache-Control: public, max-age=31536000, immutable`, `/media/*` carries `max-age=86400`, `/sw.js` carries `no-cache`
- [ ] 5.3 Manual: DevTools Network — confirm `/admin.html` does NOT carry `immutable`
- [ ] 5.4 Manual: hard reload, second navigation — confirm static bundles are served from cache (no network request)
- [ ] 5.5 Manual: disable JS in DevTools — confirm admin shell still renders with Bootstrap styling via the `<noscript>` fallback
- [x] 5.6 Checked: `v1.30.7` tag exists. Templates were restructured (new separate template files) — no unintended regression found
- [ ] 5.7 Summarize measured before/after tab-switch latency in the change journal