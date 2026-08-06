## ADDED Requirements

### Requirement: Bootstrap and FontAwesome are self-hosted
The admin UI SHALL load Bootstrap CSS/JS and FontAwesome CSS from files served under `/static/vendor/` rather than from any third-party CDN. No `<link>` or `<script>` tag in any admin template SHALL reference a hostname other than the application's own origin.

#### Scenario: Admin head references only first-party assets
- **WHEN** the rendered `/admin.html` `<head>` is inspected
- **THEN** every `<link rel="stylesheet">` and `<script src>` references a path under `/static/`
- **AND** no `https://cdn.` or `https://unpkg.com` (or similar third-party) URL appears in the head

### Requirement: Vendor assets are bundled and content-hashed by esbuild
The build process SHALL emit at least one CSS bundle and one JS bundle for the admin, each named with a content hash (e.g., `admin-vendor.<hash>.css`) so that the existing `/static/**` immutable `Cache-Control` directive applies safely.

#### Scenario: Hashed filename changes when underlying vendor version bumps
- **WHEN** the Bootstrap npm dependency is upgraded and `task build` is re-run
- **THEN** the emitted vendor CSS filename has a different `<hash>` segment than the previous build output

### Requirement: First paint is not render-blocked by vendor CSS
The admin shell's first paint SHALL not wait for the full vendor CSS bundle to load. The shell SHALL either inline the critical admin nav layout CSS or load the vendor bundle via `<link rel="preload" ... onload="this.rel='stylesheet'">` so the nav renders before the bundle finishes.

#### Scenario: Shell paints before vendor CSS loads
- **WHEN** the admin page is loaded on a throttled connection (Slow 3G profile)
- **THEN** the nav bar is visible within the first paint
- **AND** the vendor CSS resolves afterward, restyling the nav without losing its already-rendered structure

### Requirement: Service worker precaches the vendor bundles
The service worker (`sw.js`) precache manifest SHALL include the content-hashed vendor CSS and JS bundles and SHALL NOT include any third-party CDN URL.

#### Scenario: Precache manifest is vendor-only
- **WHEN** `sw.js` is inspected after a build
- **THEN** its precache list contains entries under `/static/vendor/` (or wherever the hashed bundles are emitted)
- **AND** no entry references a third-party origin