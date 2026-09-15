## ADDED Requirements

### Requirement: Templates live in files, not Go literals

The system SHALL define all server-rendered HTML in `.html` template files; Go handlers SHALL NOT contain HTML raw-string literals or inline `template.Parse(` HTML strings for page or HTMX fragment rendering.

#### Scenario: No HTML literals in handlers

- **WHEN** inspecting `handlers/htmx.go` and related HTMX handlers after the change
- **THEN** no `template.Must(template.New(...).Parse(` with an inline HTML literal remains; all HTML is loaded from template files

#### Scenario: Fragment edits via file only

- **WHEN** a developer changes an HTMX fragment's markup
- **THEN** the change is made in an `.html` file under the unified template root without modifying Go source

### Requirement: Single template loader with fail-fast parsing

The system SHALL provide a single template-loading mechanism that parses all templates at startup via `template.Must` (or equivalent) and fails fast on missing or invalid templates; handlers SHALL obtain templates from a centralized registry/lookup rather than per-file ad-hoc parsing.

#### Scenario: Missing template fails at startup

- **WHEN** the application starts and a required template file is missing or invalid
- **THEN** startup fails fast (panic or fatal error) rather than failing per-request

#### Scenario: Centralized lookup

- **WHEN** a handler renders a page or HTMX fragment
- **THEN** it does so via `ExecuteTemplate` (or equivalent) against the single shared `*template.Template` set

### Requirement: Auto-escaping preserved via html/template

The system SHALL render all templates with `html/template` (not `text/template`) so auto-escaping is preserved; no template SHALL disable escaping in a way that introduces XSS.

#### Scenario: User input is escaped

- **WHEN** a template renders user-controlled content containing HTML special characters
- **THEN** the output is auto-escaped by `html/template` (e.g. `<` becomes `&lt;`)

#### Scenario: No text/template usage for HTML

- **WHEN** inspecting template loading code
- **THEN** it imports `html/template` for HTML rendering and does not use `text/template` for HTML output

### Requirement: Embedded templates with dev disk override

The system SHALL embed default templates into the `heat-server` binary via `//go:embed` so the binary is self-contained, and SHALL allow an on-disk template directory under `server.BasePath` to override or overlay the embedded templates for development, preserving current disk-loading behavior.

#### Scenario: Binary runs without template directory

- **WHEN** `heat-server` is run from a directory that does not contain `static/templates/` on disk
- **THEN** all pages and HTMX fragments still render correctly from the embedded templates

#### Scenario: Dev disk override takes precedence

- **WHEN** a template file exists both in the embedded FS and on disk under `server.BasePath`
- **THEN** the on-disk file's content is used (disk overlays embed) so local edits are visible without rebuilding

### Requirement: Rendered output unchanged by extraction

The extraction SHALL be behavior-preserving: rendered HTML for pages and HTMX fragments SHALL remain identical (modulo insignificant whitespace normalization where documented), preserving element IDs and HTMX attributes (`hx-target`, `hx-swap`, `hx-post`, `hx-trigger`) that clients depend on.

#### Scenario: HTMX fragment golden test passes

- **WHEN** rendering each HTMX fragment (racers, tracks, quotes, seasons, teams tables and forms) with representative data before and after the extraction
- **THEN** the normalized rendered output matches the golden snapshot and `hx-target` IDs (e.g. `racer-list`, `track-list`, `quote-list`, `seasons-list`, `team-list`) are unchanged

#### Scenario: Page templates still render

- **WHEN** requesting pages that use `base.html` plus a page template (e.g. `/`, `/stats.html`, `/seasons.html`, `/trophies.html`, `/admin.html`)
- **THEN** the response is 200 with correctly composed layout and page content from the unified loader

### Requirement: Existing static/templates files reconciled without duplication

The system SHALL reconcile the 13 existing files in `static/templates/` (admin-footer.html, admin-header.html, admin.html, base.html, index.html, seasons.html, stats.html, tab-config.html, tab-drivers.html, tab-extensions.html, tab-race-day.html, tab-season.html, trophies.html) into the unified template layout without duplicating template content; `serveTemplate` and `initAdminTemplate` call sites SHALL use the single loader.

#### Scenario: No duplicate template definitions

- **WHEN** listing templates in the unified set after migration
- **THEN** each of the 13 original templates exists exactly once under the unified root and no duplicate definitions remain in `static/templates/` or elsewhere

#### Scenario: Admin tabs render via unified loader

- **WHEN** requesting admin tab fragments (race-day, season, drivers, config, extensions)
- **THEN** they are rendered from the unified `*template.Template` set rather than a separate `adminTemplate` parsed from a disjoint file list
