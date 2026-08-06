## ADDED Requirements

### Requirement: Service worker precache is generated from the vendor manifest
The build process (`build.mjs`) SHALL generate `static/sw.js` from a template (`static/sw.template.js`) by injecting a `PRECACHE_URLS` array composed of the fixed non-vendor entries (`/`, `/static/style.css`, `/static/favicon.svg`, `/static/manifest.json`) plus every path in `static/vendor/manifest.json` (`bootstrapCss`, `bootstrapJs`, `fontawesomeCss`, `adminNavCss`). `static/sw.js` SHALL NOT be hand-edited; it SHALL be regenerated on every frontend build.

#### Scenario: Build regenerates sw.js from the manifest
- **WHEN** `npm run build:frontend` is run after a vendor dependency changes
- **THEN** `static/sw.js` is overwritten with a `PRECACHE_URLS` array whose `/static/vendor/*` entries exactly match the paths in `static/vendor/manifest.json`
- **AND** no `/static/vendor/*` path in `sw.js` references a file that does not exist on disk

#### Scenario: Stale hand-edit is overwritten
- **WHEN** a contributor manually edits a `/static/vendor/*` path in `static/sw.js` and then runs `npm run build:frontend`
- **THEN** the manual edit is overwritten with the manifest-derived path
- **AND** the resulting `sw.js` matches the manifest

### Requirement: CI asserts every precache path resolves to a file on disk
The CI pipeline SHALL run a verification script (`scripts/verify-sw-precache.mjs`) after the TypeScript/frontend build that parses `static/sw.js`'s `PRECACHE_URLS`, resolves every `/static/vendor/*` path against the filesystem, and fails the build if any path is missing or if any `manifest.json` path is absent from `sw.js`.

#### Scenario: Missing vendor file fails CI
- **WHEN** `sw.js` references `/static/vendor/admin-nav.stalehash.css` but no such file exists on disk
- **THEN** the verification script exits with a non-zero status
- **AND** the CI job fails

#### Scenario: Manifest path missing from sw.js fails CI
- **WHEN** `manifest.json` declares `adminNavCss: /static/vendor/admin-nav.ca3c352da7b9.css` but `sw.js`'s `PRECACHE_URLS` does not contain that path
- **THEN** the verification script exits with a non-zero status
- **AND** the CI job fails

#### Scenario: Consistent build passes CI
- **WHEN** `sw.js`'s `/static/vendor/*` precache entries exactly match the paths in `manifest.json` and every referenced file exists on disk
- **THEN** the verification script exits with status 0
- **AND** the CI job continues

### Requirement: Service worker cache version bumps when the precache set changes
The generated `sw.js` SHALL declare a `CACHE` version string. When the precache set changes (a vendor hash changes, a vendor file is added or removed), the cache version SHALL change so deployed service workers evict the old cache on activation.

#### Scenario: Cache version differs from the broken v3
- **WHEN** the generated `sw.js` is inspected
- **THEN** the `CACHE` constant equals `heat-cache-v4` (or a later value)
- **AND** it does not equal `heat-cache-v3`

#### Scenario: Old cache is evicted on activation
- **WHEN** a deployed client with a `heat-cache-v3` cache activates the new service worker
- **THEN** the `activate` handler deletes the `heat-cache-v3` cache
- **AND** the new precache is populated from the generated `PRECACHE_URLS`

### Requirement: Generated sw.js carries a do-not-edit header
The generated `static/sw.js` SHALL begin with a comment stating it is generated from `static/sw.template.js` by `build.mjs` and must not be edited directly, so contributors are directed to the template.

#### Scenario: Header present in generated file
- **WHEN** the first lines of `static/sw.js` are inspected
- **THEN** a comment identifies the file as generated
- **AND** names `static/sw.template.js` as the source to edit
