## Why

`handlers/htmx.go` (754 lines) embeds HTML as Go raw-string literals parsed at init — e.g. `var racersTableTmpl = template.Must(template.New("racers_table").Parse(` + "`<tbody id=\"racer-list\">...`" + `))` and `racerEditFormTmpl`, with similar per-component table/form pairs for tracks, quotes, seasons, and teams. These inline templates have no syntax highlighting, linting, or formatting, bury handler logic in markup, cannot be reused as partials or overridden without recompiling, and duplicate the second templating world that already exists in `static/templates/` (13 `.html` files: admin-footer.html, admin-header.html, admin.html, base.html, index.html, seasons.html, stats.html, tab-config.html, tab-drivers.html, tab-extensions.html, tab-race-day.html, tab-season.html, trophies.html). `main.go` loads those file templates from disk at runtime via `loadCachedTemplate` / `serveTemplate` / `initAdminTemplate` / `servePage` under `server.BasePath`, while a single binary `heat-server` is produced by `task build`. The two mechanisms share no loader, no directory layout, and no reuse story; every HTMX fragment change requires editing Go source.

## What Changes

- Move all embedded HTML string literals out of `handlers/htmx.go` (and any sibling handler files that contain similar `template.Must(template.New(...).Parse(` patterns) into standalone `.html` files on disk.
- Establish a single template-loading mechanism that replaces both the ad-hoc `template.Must(...Parse(` calls in handlers and the current `static/templates/` disk-only loading in `main.go`.
- Embed templates into the `heat-server` binary via `//go:embed` so the binary remains self-contained and does not depend on the working directory at runtime; allow an on-disk override when a template directory exists under `server.BasePath` (hybrid mode) to preserve current dev behavior and hot-iteration.
- Centralize a template registry/lookup (single `*template.Template` set or named lookup) used by both page renders and HTMX fragment handlers; parse once at startup with `template.Must` so missing or invalid templates fail fast.
- Reconcile the 13 existing `static/templates/` files into the unified layout (move or reference, do not duplicate); keep rendered HTML output identical (behavior-preserving refactor).
- Guard the refactor with golden/snapshot tests of rendered HTMX fragments to catch escaping or whitespace regressions.

## Capabilities

### New Capabilities

- `server-rendered-templates`: Unified, file-based server-rendered templates embedded into the binary with a single loader, fail-fast parsing, preserved `html/template` auto-escaping, and optional dev disk override.

### Modified Capabilities

None.

## Impact

Handlers (`handlers/htmx.go` and related HTMX handlers), `main.go` template helpers (`loadCachedTemplate`, `serveTemplate`, `initAdminTemplate`), `static/templates/` layout, build (embed), and tests (new snapshot/golden tests). No user-visible behavior change; rendered HTML and HTMX `hx-target`/`hx-swap` contracts remain identical. Risk is limited to template-loading precedence and whitespace/escaping drift, mitigated by snapshot tests.
