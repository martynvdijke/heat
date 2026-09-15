## Context

Heat is a Go + Gin + Ent + HTMX app that renders both full pages and HTMX fragments server-side with `html/template`. There are two template worlds today: (1) `handlers/htmx.go` (754 lines) defines per-component templates as Go raw strings — `racersTableTmpl`, `racerEditFormTmpl`, `tracksTableTmpl`, `trackEditFormTmpl`, `quotesTableTmpl`, `quoteEditFormTmpl`, `seasonsTableTmpl`, `teamsTableTmpl`, `teamEditFormTmpl`, `seasonNewFormTmpl`, etc. — each via `template.Must(template.New(...).Parse(`...`))` at init; (2) `static/templates/` holds 13 file templates (`admin.html`, `base.html`, `index.html`, `stats.html`, `seasons.html`, `trophies.html`, `admin-header.html`, `admin-footer.html`, `tab-*.html`) loaded from disk at runtime via `main.go` helpers `loadCachedTemplate`, `serveTemplate`, `initAdminTemplate`, and `servePage` under `server.BasePath`. Static HTML pages (`static/index.html`, `static/stats.html`, `static/trophies.html`, etc.) overlap with the template files. The app ships as a single binary `heat-server` (`task build`). The goal is a single, file-based, embedded template system with no HTML literals in Go.

## Goals / Non-Goals

**Goals:**

- Move all embedded HTML literals out of Go handlers into `.html` files.
- Single template-loading mechanism for both page and HTMX fragment templates.
- `//go:embed` so `heat-server` is self-contained; optional disk override for dev preserves current `BasePath` loading.
- Centralized registry/lookup with fail-fast parsing at startup.
- Preserve `html/template` auto-escaping and keep rendered output byte-identical (modulo insignificant whitespace).
- Reconcile existing 13 `static/templates/` files without duplication.

**Non-Goals:**

- Changing rendered HTML, HTMX attributes (`hx-target`, `hx-swap`, `hx-post`), or handler behavior.
- Switching to `text/template`, introducing a new templating engine, or adding a frontend build step.
- Unifying static HTML pages and templates beyond referencing/moving (follow-up).
- Hot-reload file watching in production (dev override is manual restart or opt-in watcher later).

## Decisions

### D1: Template source — hybrid `//go:embed` with dev disk override

Embed defaults via `//go:embed` (e.g. `//go:embed templates/...` or `static/templates/...`) into an `embed.FS` parsed at startup; if a template directory exists on disk under `server.BasePath`, prefer disk files (or overlay) so `go run` and local dev see edits without rebuilding. This preserves the current disk-loading behavior during development while making deployed/relocated binaries independent of cwd. Alternatives: disk-only (keeps cwd dependence, breaks single-binary deployment), embed-only (requires rebuild for every template tweak, poor dev ergonomics). Chosen hybrid gives both properties.

### D2: File location and naming, partials reuse

Place all templates under a single root — either `templates/` at repo root or `static/templates/` retained as the canonical root — with per-domain subdirectories `fragments/racers/`, `fragments/tracks/`, `fragments/quotes/`, `fragments/seasons/`, `fragments/teams/`, plus `layouts/` (`base.html`, `admin.html`) and `partials/` (`admin-header.html`, `admin-footer.html`, `tabs/`). Each former `var ...Tmpl` becomes one file: `table.html` and `form.html` per domain (e.g. `fragments/racers/table.html`, `fragments/racers/form.html`). Define blocks (`{{define "racers_table"}}`) so fragments can be composed as partials. Existing `tab-*.html` files become `partials/tabs/*.html`. Alternatives: per-package `handlers/templates/` (splits the same concern across packages); single flat directory (does not scale, name collisions). Alternatives rejected.

### D3: Single loader with `template.Must` at startup, fail fast

One loader function (e.g. `LoadTemplates(embedFS fs.FS, basePath string) *template.Template`) parses all templates once at init via `template.Must(template.ParseFS(...))` and, when a disk dir exists, overlays with `ParseFiles`/`ParseGlob`. Registry is a single `*template.Template` set looked up by `ExecuteTemplate(w, name, data)`. Any missing or invalid template panics at startup rather than failing per-request. Alternatives: lazy per-request parsing (hides errors, slower), per-handler `Parse` calls (duplicates current fragmentation). Fail-fast is the correct default for a server.

### D4: Stay on `html/template` — never `text/template`

All templates are HTML; auto-escaping is a security invariant. The loader imports `html/template` exclusively. Alternatives: `text/template` (no escaping, XSS risk), manual escaping helpers (error-prone). Rejected.

### D5: HTMX fragment organization

Each domain gets its own subdirectory with `table.html` (tbody fragment) and `form.html` (edit/create form), preserving the current `hx-post`, `hx-target`, `hx-swap`, and element `id` contracts (`racer-list`, `track-list`, `quote-list`, `seasons-list`, `team-list`, etc.). Fragments are defined as named templates (`{{define "racers_table"}}`, `{{define "racer_form"}}`) executed via `ExecuteTemplate`. Alternatives: one large fragments file (merge conflicts, hard to navigate), client-side templating (out of scope, breaks HTMX model).

### D6: Reconciliation with the 13 existing `static/templates/` files

Move or reference the 13 files into the unified root; do not duplicate. `base.html`, `admin.html`, `admin-header.html`, `admin-footer.html`, `tab-*.html`, `index.html`, `stats.html`, `seasons.html`, `trophies.html` become `layouts/` and `partials/` members of the same `*template.Template` set as the fragments. Call sites `serveTemplate` (`base.html` + page) and `initAdminTemplate` (admin + tabs) switch to the unified loader. Static HTML pages (`static/index.html` etc.) that duplicate templates are noted for a follow-up unification; this change only ensures templates are not duplicated. Alternative — keep two roots — is exactly the problem being fixed.

## Risks / Trade-offs

- HTML escaping or whitespace changes altering rendered output or breaking HTMX `hx-target` IDs → Mitigate with golden/snapshot tests of every fragment's rendered output before/after, comparing normalized and exact HTML.
- Embed vs disk precedence bugs (stale embedded content shadowing disk edits, or disk partial shadowing breaking in prod) → Mitigate by documenting precedence (disk overlays embed), logging which source was used at startup, and testing both modes.
- Deployment where binary is moved and `BasePath` templates are absent → Mitigated because embed is the fallback; binary remains self-contained.
- Template parse errors now fail the process at startup instead of per-request → Intended; add CI check that `go vet` / `go test` exercises template loading.
- Renaming/moving 13 existing templates may break relative references → Mitigate with a single commit moving files and updating `ExecuteTemplate` names, verified by existing page tests.

## Migration Plan

1. Add `//go:embed` FS and new loader (`LoadTemplates` / registry) alongside existing code; wire it to log parsed template names at startup; keep old `var ...Tmpl` vars in place.
2. Move HTMX templates in batches by domain (racers → tracks → quotes → seasons → teams), each as one `.html` file pair; update the corresponding handler to `ExecuteTemplate` from the registry; snapshot-test rendered output against golden files captured before the move.
3. Migrate the 13 `static/templates/` files into the unified layout (layouts/partials) and update `serveTemplate`/`initAdminTemplate` to use the single loader; remove `loadCachedTemplate` disk-only path once all callers are switched (or keep as thin wrapper over registry during transition).
4. Unify or de-duplicate `static/*.html` overlap in a follow-up if desired (out of scope for behavior-preserving extraction).
5. Delete the raw-string `var ...Tmpl` definitions only after each fragment's snapshot matches.

Rollback: keep old raw-string template vars until their replacement's snapshot test passes; any domain can be reverted by restoring the var and reverting the handler to `varTmpl.Execute`. The commit history is batched per domain so revert is per-domain. `OIDC_ENABLED`-style toggle is not needed — this is a pure refactor with output-equivalence tests.

## Open Questions

- Single `templates/` root at repo top vs retaining `static/templates/` as the canonical root (and whether `static/` should only serve static assets).
- Should `static/*.html` pages (`static/index.html`, `static/stats.html`, `static/trophies.html`) and `static/templates/*.html` be unified into one set, or should static pages remain plain files served by `servePage`?
- Dev hot-reload expectations: is restart-on-change sufficient, or should the dev server watch template files and re-parse without restart?
