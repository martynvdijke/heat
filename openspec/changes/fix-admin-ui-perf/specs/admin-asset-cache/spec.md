## ADDED Requirements

### Requirement: Static asset route cache headers
The system SHALL serve all files under `/static/**` with an HTTP response header `Cache-Control: public, max-age=31536000, immutable` because the esbuild-produced bundles are content-hashed.

#### Scenario: Repeat navigation reuses cached bundle
- **WHEN** a browser that has already fetched `/static/admin.js?v=abc123` navigates back to `/admin.html`
- **THEN** the browser MUST NOT send a conditional request for `/static/admin.js?v=abc123` and MUST serve the bundle from cache

#### Scenario: First fetch captures a long cache entry
- **WHEN** a browser fetches any path under `/static/**` for the first time
- **THEN** the response MUST contain `Cache-Control: public, max-age=31536000, immutable`

### Requirement: Media route cache headers
The system SHALL serve all files under `/media/**` with an HTTP response header `Cache-Control: public, max-age=86400` because media content may change but is safe to cache for one day.

#### Scenario: Media is cached for a day
- **WHEN** a browser fetches any path under `/media/**`
- **THEN** the response MUST contain `Cache-Control: public, max-age=86400`

#### Scenario: Media is revalidated after a day
- **WHEN** a browser re-requests a media path after the max-age has expired
- **THEN** the browser MUST send a conditional request and update its cached copy if the server returns 200

### Requirement: Service worker route cache headers
The system SHALL serve `/sw.js` with an HTTP response header `Cache-Control: no-cache` so the service worker is always revalidated by the browser.

#### Scenario: Service worker is revalidated every navigation
- **WHEN** a browser navigates to any page controlled by the service worker
- **THEN** the browser MUST fetch `/sw.js` from the network (no-cache) and update the registered service worker if the byte contents differ

### Requirement: Admin HTML is not long-cached
The system SHALL NOT apply the immutable 1y `Cache-Control` directive to the `/admin.html` route or any HTML template response. HTML MAY carry a short revalidation directive but MUST NOT be cached immutable.

#### Scenario: Admin HTML always revalidates
- **WHEN** a browser re-requests `/admin.html`
- **THEN** the response MUST NOT contain `immutable`