## ADDED Requirements

### Requirement: Per-tab fragment endpoints exist under admin auth
The server SHALL expose four `GET` endpoints: `/api/html/admin/race-day`, `/api/html/admin/season`, `/api/html/admin/drivers`, `/api/html/admin/config`. Each SHALL be protected by the existing admin auth middleware (`middleware.AuthMiddleware`) and the existing CSRF middleware.

#### Scenario: Unauthenticated request is rejected
- **WHEN** an unauthenticated client issues `GET /api/html/admin/drivers`
- **THEN** the server responds with HTTP 401 (or the same redirect-to-login response the existing `/api/html/*` admin handlers return for unauthenticated requests)

#### Scenario: Authenticated request returns the tab fragment
- **WHEN** an authenticated admin session issues `GET /api/html/admin/drivers`
- **THEN** the server responds with HTTP 200 and a `Content-Type: text/html; charset=utf-8` body containing the Drivers tab's panes and modals only

### Requirement: Endpoint returns only the requested tab's content
A fragment response SHALL include only the panes and modals belonging to the requested tab. Panes and modals for other tabs SHALL NOT appear in the response.

#### Scenario: Drivers response excludes Season content
- **WHEN** the `/api/html/admin/drivers` body is parsed
- **THEN** no element with `data-tab-pane="season"` is present
- **AND** no Season-only modal (e.g., `#seasonsModal`) is present
- **AND** Drivers-only modals (e.g., `#racerModal`, `#teamModal`, `#quoteModal`) ARE present

### Requirement: Unknown tab id is rejected
A `GET /api/html/admin/{tab}` request with an unrecognized `{tab}` value SHALL respond with HTTP 404 and an empty or error-body response.

#### Scenario: Unknown tab id 404s
- **WHEN** an authenticated client issues `GET /api/html/admin/nonexistent`
- **THEN** the server responds with HTTP 404

### Requirement: Fragment responses are never cached
Fragment responses SHALL include `Cache-Control: no-store` so a stale pane never ships mid-race.

#### Scenario: Fragment carries no-store
- **WHEN** any `/api/html/admin/{tab}` response's headers are inspected
- **THEN** the `Cache-Control` header equals `no-store`
- **AND** no `max-age` directive is present

### Requirement: Active tab server-rendered into /admin.html shares the fragment handler
The active (Race Day by default) tab's content rendered eagerly into `/admin.html` SHALL be produced by the same handler/template that serves `/api/html/admin/race-day`, so the eager and lazy paths cannot drift.

#### Scenario: Eager and lazy Race Day HTML are template-identical
- **WHEN** the Race Day portion of the `/admin.html` response body is compared to the body of `GET /api/html/admin/race-day`
- **THEN** the HTML fragments are byte-identical (modulo any CSRF token placeholders substituted by the auth context)