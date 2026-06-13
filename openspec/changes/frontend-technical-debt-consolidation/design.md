## Context

The frontend consists of 14+ static HTML pages served by a Go/Gin backend. The `servePage()` function reads HTML files, replaces `{{VERSION}}`, and serves them. Currently no HTML templating — every page independently declares its own `<head>`, nav, header, and footer. TypeScript compiles file-by-file via `tsc` without bundling. CSS is split between a shared `style.css` (882 lines) and per-page `<style>` blocks.

The backend was recently restructured (monolith splitting in `fix-race-report-and-split-monoliths`). The frontend hasn't received similar structural attention.

Existing changes `admin-controller-ui-cleanup` and `add-umami-analytics` are in progress and may conflict with HTML restructuring — sequencing matters.

## Goals / Non-Goals

**Goals:**
- Eliminate duplicated HTML structure across all pages via Go `html/template` partials
- Consolidate all page-specific inline CSS into `static/style.css` with unified CSS custom properties
- Remove embedded GeoJSON data (500+ lines) from `ts/index.ts` into proper data storage
- Split `static/admin.html` (1316 lines) into manageable pieces
- Add a frontend build pipeline (esbuild) for bundling, minification, and code sharing
- Migrate all inline JavaScript into proper TypeScript modules with consistent error handling
- Clean up `interface{}` / loose types in Go backend code

**Non-Goals:**
- Rewriting the frontend in a SPA framework (React, Vue, etc.)
- Changing the visual design or theme
- Adding new features beyond the consolidation work
- Full separation of admin into completely independent pages (partial-based approach within single file is sufficient)
- Server-side rendering of dynamic content (current client-rendered approach works)

## Decisions

### 1. HTML Template System: Go `html/template` with inheritance

**Decision:** Use Go's `html/template` package with `define`/`template` blocks to create a base layout and page-specific templates.

**Rationale:**
- Already have Go on the server — no new dependency
- `html/template` supports template composition via `{{define "name"}}` and `{{template "name" .}}`
- Can create `base.html` with the shared chrome (head, nav, header, footer) and have each page define only its content blocks
- Minimal change to the existing `servePage()` pattern — adapt it to use `template.ParseFiles()` instead of `os.ReadFile()`
- Avoids adding a separate build step for HTML (like Eleventy, Hugo, etc.)

**Alternatives considered:**
- Static site generator (Hugo): Overkill for a small app, adds build complexity
- Server-side includes (SSI): Requires web server config changes, less flexible
- Client-side includes (JS fetch): Worse performance, SEO issues, adds JS dependency
- Keep as-is: Continues accruing debt, every new page duplicates the same markup

### 2. CSS Consolidation: Extract all inline styles into style.css

**Decision:** Move every inline `<style>` block into `static/style.css` under section comments. Normalize all color values to use the canonical `--primary`, `--racing-green`, `--background` variables.

**Rationale:**
- Single source of truth for all styles
- Eliminates confusion between `--heat-red` (admin), `--primary` (public pages), and raw hex values
- Reduces CSS payload (no duplicated rules across inline blocks)
- Simple find-and-move approach, low risk

### 3. GeoJSON Data: Static JSON file served via API endpoint

**Decision:** Extract GeoJSON into a static JSON file at `static/data/tracks-geojson.json`, served via a new `GET /api/tracks/geojson` endpoint that reads from the database `tracks.geojson` column.

**Rationale:**
- The `tracks` table already has a `geojson` column — it's the canonical source
- A new API endpoint is consistent with the existing REST pattern
- Removes 500+ lines of hardcoded data from compiled JS
- Frontend fetches it like any other API data — no change to data flow
- Adding a JSON file as a fallback avoids a DB hit on every page load (can cache)

**Alternatives considered:**
- Serve static JSON file directly: Simpler but bypasses the DB as source of truth
- Keep in TypeScript: Maintains current technical debt

### 4. Admin Split: Go template partials within single page

**Decision:** Use Go `html/template` partials (`{{define "racersTab"}}` etc.) rather than splitting into separate HTML files. Each tab pane becomes a named template in a separate file (e.g., `static/admin-racers.html`), included by the main admin template.

**Rationale:**
- Least disruptive to the existing admin UX (no navigation model change)
- HTMX is already partially used — could complement but not replace the tab structure
- Keeps URL structure unchanged
- Each tab is independently editable without touching the 1316-line file

### 5. Build Pipeline: esbuild

**Decision:** Add esbuild as the frontend bundler, replacing `tsc` for compilation. Configure it to:
- Bundle shared modules (toast.ts, i18n.ts) into each page's JS
- Minify output
- Generate sourcemaps for development

**Rationale:**
- 10-100x faster than `tsc` for TypeScript compilation
- Native bundling support — no extra config for code sharing
- Handles TypeScript, minification, sourcemaps out of the box
- Can keep `tsc` for type-checking only (faster dev loop)
- Small dependency (~5MB binary, no runtime dependencies)

**Alternatives considered:**
- Vite: Great for SPAs but overkill for static HTML pages with isolated TS files
- Webpack: Heavy configuration, slow, unnecessary for this scale
- Keep tsc only: No bundling means toast.ts must be loaded as a separate script tag on every page, no code sharing

### 6. Inline JS Migration: New TS modules for each page

**Decision:** Create `ts/login.ts`, `ts/driver.ts`, etc. for each page with inline JS. Move inline code into these files, adopting the same patterns as existing TS files (proper error handling, strict types).

**Rationale:**
- Consistent with existing architecture
- TypeScript catches errors during development
- esbuild bundles these with shared modules automatically
- Remove `onclick` HTML attributes in favor of `addEventListener` in TS — better separation of concerns

### 7. Backend Type Cleanup: Targeted audit

**Decision:** Audit handlers that use `interface{}` or `any` in function signatures and return types. Replace with proper Go types where the actual shape is known.

**Rationale:**
- Improves compile-time safety
- Makes the API surface self-documenting
- Low-risk targeted changes

## Risks / Trade-offs

- **[Risk] HTML template migration breaks page rendering** → Mitigation: Migrate one page at a time, verify each visually and via Playwright tests before proceeding to the next
- **[Risk] esbuild changes break existing TS compilation** → Mitigation: Keep `tsc` for type-checking, use esbuild only for bundling. Run both in CI initially, then phase out tsc
- **[Conflict] admin-controller-ui-cleanup modifies admin.html and controller.html** → Mitigation: Sequence this change after that one completes, or rebase if both are in progress
- **[Risk] GeoJSON API endpoint adds latency to home page load** → Mitigation: Cache the response in-memory on the server with a 5-minute TTL; the data rarely changes
- **[Trade-off] Go templates require server restart for HTML changes** → Already the case with the current `os.ReadFile()` approach — no regression

## Open Questions

- Should GeoJSON be fetched from the API on every page load, or should the frontend cache it in localStorage?
- How should versioning work for esbuild output files (hash-based filenames for cache busting vs. manual version bumps)?
- Should the admin page remain a single URL with tab partials, or split into separate routes (`/admin/racers`, `/admin/tracks`, etc.)?
