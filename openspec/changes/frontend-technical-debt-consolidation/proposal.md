## Why

The frontend codebase has accumulated significant structural debt across 14+ HTML pages — duplicated markup, inconsistent styling patterns, embedded data blobs, monolithic files, and no build pipeline. Each issue compounds maintenance cost and makes every UI change touch multiple files. Fixing these now reduces friction for all future frontend work.

## What Changes

### 1. Eliminate Duplicated HTML Templates
- Extract shared header, nav, footer, and `<head>` sections from all HTML pages into reusable partials
- Use Go server-side template includes or a lightweight HTML templating approach
- All 14+ pages get consistent structure from a single source of truth

### 2. Unify CSS Into Shared Stylesheet
- Move inline `<style>` blocks from TV, Pitboard, Spectator, Player, Replay, Driver, and other pages into `static/style.css`
- Replace page-specific CSS variables (`--heat-red`, etc.) with the canonical custom properties from `:root`
- Remove duplicate style rules that exist in both inline blocks and style.css

### 3. Extract Hardcoded GeoJSON Data
- Move the 500+ lines of track GeoJSON coordinates from `ts/index.ts` into proper data storage
- Options: JSON data files loaded at runtime, or a new API endpoint serving GeoJSON from the database
- Remove the now-unnecessary embedded coordinate data from compiled JS

### 4. Split Admin Monolith
- Break `static/admin.html` (1316 lines) into logical pieces
- Either extract tab panes into partials, or split into separate admin sub-pages
- Ensure no regression in the 16-tab admin experience (which is being reorganized by `admin-controller-ui-cleanup`)

### 5. Add Frontend Build Pipeline
- Introduce a bundler (esbuild or vite) to handle TypeScript compilation, minification, and code sharing
- Enable module sharing so toast.ts and i18n.ts are available to all pages without duplication
- Generate versioned/hashed output filenames for cache busting

### 6. Migrate Inline JavaScript to TypeScript Modules
- Move inline JS from `login.html`, `driver.html`, `tv.html`, and `pitboard.html` into proper `.ts` files
- Apply consistent error handling and TypeScript strict mode across all frontend code
- Remove `onclick` attributes from HTML where possible, using proper event listeners

### 7. Clean Up Backend Type Loose Ends
- Replace `interface{}` and `any` types in Go backend with proper typed structs where found
- Target specific areas in handlers, racing package, and models

## Capabilities

### New Capabilities
- `html-partial-system`: Reusable HTML partials for headers, nav, footers, and meta tags shared across all pages
- `css-style-unification`: Consolidated, single-source-of-truth stylesheet with all page styles extracted from inline blocks
- `geo-json-data-service`: Track GeoJSON data served from a proper data source instead of embedded in client code
- `frontend-build-pipeline`: Automated TypeScript bundling, minification, and asset versioning
- `inline-js-elimination`: All JavaScript moved from HTML attributes and inline `<script>` blocks into proper TypeScript modules

### Modified Capabilities
- `internal-code-quality`: Extended scope to include frontend code quality standards beyond the existing backend-focused spec

## Impact

- **Frontend (HTML)**: Every `.html` file in `static/` — structural changes to use shared partials instead of duplicated markup
- **Frontend (CSS)**: `static/style.css` gains 200-400 lines from extracted inline styles; per-page `<style>` blocks removed
- **Frontend (TypeScript)**: New files for extracted inline JS; `ts/index.ts` loses 500+ lines of embedded data; new build config files
- **Backend (Go)**: New or modified handler for GeoJSON data serving; type cleanup in handlers/ and racing/ packages
- **Build System**: New bundler dependency (esbuild or vite), updated `Taskfile.yml`, updated `package.json` scripts
- **Database**: May add GeoJSON storage table (optional, alternative is static JSON files)
