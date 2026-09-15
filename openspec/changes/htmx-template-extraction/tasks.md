## 1. Scaffolding and loader

- [ ] 1.1 Add `//go:embed` FS and single loader `LoadTemplates(embedFS fs.FS, basePath string) *template.Template` that parses embedded templates with `html/template` via `template.Must(template.ParseFS(...))` and overlays disk files when `server.BasePath` template dir exists; centralize registry/lookup (single `*template.Template` set, `ExecuteTemplate` by name).
- [ ] 1.2 Wire loader into `main.go` startup (replace/augment `loadCachedTemplate` / `initAdminTemplate` call sites); log parsed template names and whether disk override was applied; verify `heat-server` runs from a temp dir with no `static/templates/` on disk (embed fallback).
- [ ] 1.3 Capture golden snapshots of current rendered HTMX fragments (racers/tracks/quotes/seasons/teams tables + forms) and page renders (index/stats/seasons/trophies/admin tabs) as test fixtures for equivalence checks.

## 2. Extract HTMX fragment templates

- [ ] 2.1 Extract racers templates: move `racersTableTmpl` and `racerEditFormTmpl` HTML into `templates/fragments/racers/table.html` and `form.html` (or `static/templates/fragments/racers/...` if retaining that root); update `handlers/htmx.go` racers handlers to `ExecuteTemplate`; snapshot-test output matches golden.
- [ ] 2.2 Extract tracks templates: move `tracksTableTmpl` and `trackEditFormTmpl` into `fragments/tracks/table.html` and `form.html`; update handlers; snapshot-test.
- [ ] 2.3 Extract quotes templates: move `quotesTableTmpl` and `quoteEditFormTmpl` into `fragments/quotes/table.html` and `form.html`; update handlers; snapshot-test.
- [ ] 2.4 Extract seasons templates: move `seasonsTableTmpl` and `seasonNewFormTmpl` into `fragments/seasons/table.html` and `form.html`; update handlers; snapshot-test.
- [ ] 2.5 Extract teams templates: move `teamsTableTmpl` and `teamEditFormTmpl` into `fragments/teams/table.html` and `form.html`; update handlers; snapshot-test.

## 3. Unify existing file templates

- [ ] 3.1 Move/reconcile the 13 `static/templates/` files (admin-footer.html, admin-header.html, admin.html, base.html, index.html, seasons.html, stats.html, tab-config.html, tab-drivers.html, tab-extensions.html, tab-race-day.html, tab-season.html, trophies.html) into unified layout `layouts/` + `partials/tabs/` (no duplication); update `serveTemplate` and `initAdminTemplate` to use the single loader.
- [ ] 3.2 Remove duplication/overlap with `static/*.html` pages where applicable (reference unified templates instead of separate static files); keep `servePage` only for truly static pages (tv.html, pitboard.html, replay.html, player.html, spectator.html, driver.html, login.html, setup.html, etc.).
- [ ] 3.3 Delete raw-string `var ...Tmpl` definitions from `handlers/htmx.go` only after each fragment's snapshot matches; remove any remaining ad-hoc `template.Must(...Parse(` inline HTML.

## 4. Verification and cleanup

- [ ] 4.1 Run golden/snapshot tests for all fragments and pages; verify `hx-target`/`hx-swap`/`hx-post` IDs and `html/template` auto-escaping are preserved; test both embed-only (no disk dir) and disk-override modes.
- [ ] 4.2 Run `task pre-push` (gofmt, go vet + govulncheck, go tests, TypeScript compile, `heat-server` build) and manual smoke: `go run .` and relocated binary both serve pages and HTMX fragments correctly.
- [ ] 4.3 Update docs/comments that reference `loadCachedTemplate`/`initAdminTemplate` or inline templates to describe the unified `//go:embed` + registry approach.
