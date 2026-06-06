## ADDED Requirements

### Requirement: Admin panel has a "Logs" tab

The admin panel SHALL include a "Logs" tab in the navigation bar alongside existing tabs (Race Setup, Racers, Tracks, etc.) with a log icon. When activated, it SHALL display a log viewer pane that shows structured application logs.

#### Scenario: Logs tab exists in admin nav
- **WHEN** the admin user opens `/admin.html`
- **THEN** a "Logs" tab SHALL be visible in the tab navigation bar with a fa-list icon

#### Scenario: Logs tab is not the default active tab
- **WHEN** the admin user opens `/admin.html`
- **THEN** the Logs pane SHALL NOT be the default active pane (default remains Race Setup)

#### Scenario: Clicking Logs tab shows log viewer pane
- **WHEN** the admin user clicks the "Logs" tab
- **THEN** the log viewer pane SHALL appear with controls for filtering, refresh, and a table of log entries

### Requirement: Log viewer displays paginated log entries

The log viewer SHALL display application log entries in a table with columns: Timestamp, Level, Module, Message. Entries SHALL be paginated with 50 entries per page. The viewer SHALL auto-refresh every 5 seconds when the tab is active.

#### Scenario: Log entries appear on tab activation
- **WHEN** the Logs tab is activated
- **THEN** the most recent 50 log entries SHALL be fetched from `/api/admin/logs` and displayed

#### Scenario: Auto-refresh polls for new logs
- **WHEN** the Logs tab is active
- **THEN** new log entries SHALL be fetched every 5 seconds via polling
- **WHEN** the Logs tab is deactivated (another tab selected)
- **THEN** polling SHALL stop

#### Scenario: Pagination controls are visible
- **WHEN** the log viewer displays entries
- **THEN** pagination controls (Previous/Next) SHALL be shown when there are more entries than the current page

### Requirement: Log viewer supports filtering

The log viewer SHALL provide controls to filter displayed logs by:
- **Level**: Dropdown with options All, DEBUG, INFO, WARN, ERROR
- **Module**: Dropdown populated from available modules, with an "All" option
- A search/filter button to apply filters

#### Scenario: Filter by level
- **WHEN** the user selects "ERROR" in the level filter and clicks Apply
- **THEN** only log entries with level "ERROR" SHALL be shown (url parameter `?level=ERROR`)

#### Scenario: Filter by module
- **WHEN** the user selects "email" in the module filter and clicks Apply
- **THEN** only log entries from the "email" module SHALL be shown (url parameter `?module=email`)

#### Scenario: Combined filters
- **WHEN** the user selects level "WARN" and module "ai"
- **THEN** only entries matching BOTH filters SHALL be shown

### Requirement: Log viewer has a manual refresh button

The log viewer SHALL include a "Refresh" button that fetches the latest entries immediately without waiting for the auto-refresh interval.

#### Scenario: Manual refresh
- **WHEN** the user clicks the "Refresh" button
- **THEN** the latest log entries SHALL be fetched immediately

### Requirement: API endpoint GET /api/admin/logs

The system SHALL provide a paginated, filterable API endpoint for fetching logs at `GET /api/admin/logs` (authenticated, under the admin group).

Supported query parameters:
- `page` (int, default 1): Page number
- `pageSize` (int, default 50): Entries per page
- `level` (string, optional): Filter by level (DEBUG, INFO, WARN, ERROR)
- `module` (string, optional): Filter by module name

Response format:
```json
{
  "entries": [
    {
      "id": 1,
      "timestamp": "2026-06-06T12:00:00Z",
      "level": "WARN",
      "module": "email",
      "message": "SMTP connection timeout",
      "data": null
    }
  ],
  "total": 142,
  "page": 1,
  "pageSize": 50
}
```

#### Scenario: Fetch logs with default pagination
- **WHEN** a GET request is sent to `/api/admin/logs`
- **THEN** the response SHALL contain the first 50 log entries ordered by timestamp descending

#### Scenario: Fetch logs with level filter
- **WHEN** a GET request is sent to `/api/admin/logs?level=ERROR`
- **THEN** only entries with level "ERROR" SHALL be returned

#### Scenario: Fetch logs with module filter
- **WHEN** a GET request is sent to `/api/admin/logs?module=email`
- **THEN** only entries from the "email" module SHALL be returned

#### Scenario: Unauthenticated request returns 401
- **WHEN** an unauthenticated request is sent to `/api/admin/logs`
- **THEN** the response SHALL be HTTP 401 Unauthorized

### Requirement: API endpoint GET /api/admin/logs/modules

The system SHALL provide an endpoint at `GET /api/admin/logs/modules` that returns a list of distinct module names found in the logs.

#### Scenario: Fetch module list
- **WHEN** a GET request is sent to `/api/admin/logs/modules`
- **THEN** the response SHALL be a JSON array of unique module names, e.g., `["email", "ai", "backup", "auth", "race"]`
