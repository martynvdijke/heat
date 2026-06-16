# E-Ink Mode Tasks

## 1. Backend: Data model and DB

- [x] 1.1 Add `EInkEnabled bool` field to `PageData` struct in `main.go`
- [x] 1.2 Add `eink_settings` table creation to `db/init.go`
- [x] 1.3 Add `SeedEInkSettings()` to seed default (disabled) in `db/init.go`
- [x] 1.4 Add `EInkSettings` model to `models/models.go`

## 2. Backend: API and handler

- [x] 2.1 Create `handlers/eink_settings.go` with GET/POST endpoints following existing settings patterns
- [x] 2.2 Register `/api/eink-settings` routes and admin auth in `main.go`
- [x] 2.3 Update `serveTemplate()` to query e-ink setting and pass `EInkEnabled` to template

## 3. Frontend: CSS

- [x] 3.1 Create `static/eink.css` with high-contrast black-on-white overrides
- [x] 3.2 Add rule: disable all CSS transitions, animations, shadows, filters in eink mode
- [x] 3.3 Add rule: large touch targets (48px/56px) for interactive elements
- [x] 3.4 Add rule: simplified tables (solid borders, no alternating row colors)
- [x] 3.5 Add rule: text-based flag indicator styles (text labels instead of colored circles)
- [x] 3.6 Add rule: larger font sizes for distance readability
- [x] 3.7 Add rule: simplified driver view styles

## 4. Frontend: JS toggle

- [x] 4.1 Create `ts/eink.ts` with init function: check URL param, cookie, toggle class
- [x] 4.2 Add manual toggle button and handler with icon swap
- [x] 4.3 Integrate with admin-enforced setting (respect server-side toggle)

## 5. Frontend: Template integration

- [x] 5.1 Update `static/templates/base.html` to include `eink.css` and conditionally add `.eink-mode` class
- [x] 5.2 Add e-ink toggle button to header navigation in base template
- [x] 5.3 Add e-ink settings pane to admin settings UI (`admin-settings-panes.html`)

## 6. Build and integration

- [x] 6.1 Add `ts/eink.ts` to esbuild build config (entry point if needed)
- [x] 6.2 Verify `task build:frontend` succeeds
- [x] 6.3 Verify Go build succeeds

## 7. E2E tests

- [x] 7.1 Add Playwright test: verify `?eink=1` URL param activates e-ink mode
- [x] 7.2 Add Playwright test: verify toggle button works
- [x] 7.3 Add Playwright test: verify admin toggle persists and affects all pages
- [x] 7.4 Add Playwright test: verify normal mode is unaffected by eink.css
