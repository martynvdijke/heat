# admin-non-blocking-css Specification

## Purpose
TBD - created by archiving change fix-admin-ui-perf. Update Purpose after archive.
## Requirements
### Requirement: Non-blocking CSS for first paint
The admin HTML document SHALL load Bootstrap CSS and FontAwesome CSS without blocking first paint of the admin shell. The CSS SHALL be referenced using `<link rel="preload" as="style" ... onload="this.rel='stylesheet'">` (or an equivalent non-blocking pattern) and MUST provide a `<noscript>` fallback with a normal `<link rel="stylesheet">` for clients without JavaScript.

#### Scenario: Admin first paint is not blocked on CSS download
- **WHEN** a browser with JavaScript enabled navigates to `/admin.html` on a cold cache
- **THEN** the navigation request MUST reach `domContentLoaded` without waiting for the Bootstrap CSS `onload` event

#### Scenario: No-JS clients still receive styling
- **WHEN** a browser with JavaScript disabled navigates to `/admin.html`
- **THEN** the document MUST include a `<noscript><link rel="stylesheet" href="<bootstrap-url>"></noscript>` and a corresponding FontAwesome fallback, and the admin shell MUST render with full styling

### Requirement: CDN URLs unchanged
The CSS preload/load SHALL target the exact same CDN URLs and version strings that the previous render-blocking `<link>` tags used. No CDN migration, no self-hosting, no version bumps occur in this change.

#### Scenario: Bootstrap URL preserved
- **WHEN** the admin HTML is rendered
- **THEN** the Bootstrap CSS URL MUST be `https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css`

#### Scenario: FontAwesome URL preserved
- **WHEN** the admin HTML is rendered
- **THEN** the FontAwesome CSS URL MUST be `https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.7.2/css/all.min.css`

### Requirement: Tab pane transition capped at 200ms
The system SHALL cap any CSS transition or animation applied to admin tab panes (`.tab-pane`) at a duration no greater than 200ms. The cap SHALL be enforced via a CSS override loaded after Bootstrap's stylesheet.

#### Scenario: Switching top-nav or side-nav tabs completes promptly
- **WHEN** a user clicks a top-nav or side-nav tab in `/admin.html`
- **THEN** the visible transition on the target `.tab-pane` MUST complete in 200ms or less

#### Scenario: Bootstrap default fade is not slow
- **WHEN** a `.tab-pane` with a `fade` class becomes the active pane
- **THEN** the opacity transition duration MUST be 150ms (Bootstrap's default) or shorter, never seconds

### Requirement: Dead duplicate admin index removed
The system SHALL NOT ship a static `static/admin.html` file. Only the Go-template-served admin document exists. Any service-worker precache list MUST NOT reference `static/admin.html`.

#### Scenario: Static duplicate is gone
- **WHEN** the repository is inspected after this change
- **THEN** the file `static/admin.html` MUST NOT exist

#### Scenario: Service worker does not precache the deleted file
- **WHEN** `sw.js` is inspected after this change
- **THEN** its precache manifest MUST NOT contain an entry whose URL resolves to the deleted `static/admin.html`

