# E-Ink Mode Design

## Overview

E-ink mode adapts HEAT's UI for low-power, high-contrast e-ink displays (or any display where readability in direct sunlight matters). The mode is activated via URL parameter (`?eink=1`) with cookie persistence, and optionally enforced globally via admin settings.

## Architecture

```
User request → Cookie/URL check → Add `.eink-mode` class to `<body>` → `eink.css` overrides take effect
                 ↓
         Admin can set global flag → Server passes `EInkEnabled` to template → Base template adds class server-side
```

### Layers

1. **CSS Layer** (`static/eink.css`): Single stylesheet with all overrides scoped under `.eink-mode` body class. Loaded on all pages (inactive by default). No changes to existing `style.css` needed.

2. **JS Layer** (`ts/eink.ts`): Toggle logic — reads URL param, manages cookie, toggles class on `<body>`, provides manual toggle button. Bundled into page-specific JS via esbuild.

3. **Backend Layer** (Go): Minimal changes — add `EInkEnabled bool` to `PageData` struct, add `eink_settings` DB table + handler for admin toggle, seed init.

4. **Template Layer**: Base template conditionally adds `.eink-mode` class to `<body>` and includes `eink.css`. Admin settings tab gets an e-ink toggle pane.

## CSS Strategy

All e-ink overrides in `static/eink.css`, each rule prefixed with `.eink-mode`:

```css
/* High contrast palette */
.eink-mode,
.eink-mode body,
.eink-mode .container { background: #fff !important; color: #000 !important; }
.eink-mode .text-muted { color: #333 !important; }

/* Remove all motion/effects */
.eink-mode * { animation: none !important; transition: none !important; }
.eink-mode * { box-shadow: none !important; text-shadow: none !important; }
.eink-mode * { backdrop-filter: none !important; filter: none !important; }

/* Large touch targets */
.eink-mode .btn,
.eink-mode button,
.eink-mode a { min-width: 48px; min-height: 48px; }
.eink-mode .race-control-btn { min-width: 56px; min-height: 56px; }

/* Table simplification */
.eink-mode table tr { border-bottom: 1px solid #ccc; }
.eink-mode .table-striped tr { background: #fff !important; }

/* Flag indicators */
.eink-mode .flag-indicator { border: 2px solid #000; }
.eink-mode .flag-active { background: #000; color: #fff; }
```

## JS Toggle Mechanism

- `ts/eink.ts` exports `initEInkMode()` called from each page's entry point
- On load: checks `localStorage.getItem('eink')` and `new URLSearchParams(window.location.search).get('eink')`
- If URL param `?eink=1`: saves to localStorage, adds class
- Manual toggle button in nav area
- Admin-enforced flag (from `pageData`) is checked server-side: if server says eink is forced, ignore local preference

## Backend Changes

### `PageData` struct (main.go)
```go
type PageData struct {
    Version     string
    EInkEnabled bool
}
```

### Database
New table `eink_settings`:
```sql
CREATE TABLE IF NOT EXISTS eink_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0
);
```

### API
- `GET /api/eink-settings` → returns `{enabled: bool}`
- `POST /api/eink-settings` → saves `{enabled: bool}` (admin only)

### Handler
New `handlers/eink_settings.go` following existing patterns.

### Template Data Flow
1. `GET /admin` or any template page calls `serveTemplate`
2. `serveTemplate` queries `eink_settings` or reads cached value
3. Sets `PageData.EInkEnabled`
4. Base template checks `{{if .EInkEnabled}}` → adds `.eink-mode` class to `<body>` or sets data attribute

## Test Strategy

- **E2E (Playwright)**: Test `?eink=1` param activates mode, test toggle button, test admin toggle persists
- **Visual**: Screenshot comparison for key pages in e-ink mode
- **CSS**: No visual regressions in normal mode (eink.css is opt-in)
