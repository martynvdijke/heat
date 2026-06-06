## ADDED Requirements

### Requirement: Log verbosity is configurable per module

The system SHALL allow configuring log verbosity level per module. Levels are: DEBUG, INFO, WARN, ERROR. Unconfigured modules SHALL default to WARN. The configuration SHALL be stored in a `log_settings` table and be mutable at runtime via API.

#### Scenario: Default verbosity is WARN
- **WHEN** a log call is made from a module with no explicit verbosity setting
- **THEN** the log entry SHALL be recorded only if its level is WARN or higher (WARN, ERROR)

#### Scenario: Module configured to DEBUG shows all logs
- **WHEN** module "email" is configured with level "DEBUG"
- **THEN** all log calls from module "email" at any level SHALL be recorded (DEBUG, INFO, WARN, ERROR)

#### Scenario: Module configured to ERROR shows errors only
- **WHEN** module "email" is configured with level "ERROR"
- **THEN** only ERROR-level log calls from module "email" SHALL be recorded

### Requirement: API endpoint GET /api/admin/log-settings

The system SHALL provide an authenticated endpoint to retrieve current log verbosity settings.

Response format:
```json
{
  "settings": [
    {"module": "email", "level": "DEBUG"},
    {"module": "ai", "level": "WARN"}
  ],
  "default": "WARN"
}
```

#### Scenario: Fetch log settings
- **WHEN** a GET request is sent to `/api/admin/log-settings`
- **THEN** the response SHALL contain an array of module-level overrides and a default level

### Requirement: API endpoint POST /api/admin/log-settings

The system SHALL provide an authenticated endpoint to save log verbosity settings. The request body SHALL be an array of `{module, level}` objects.

#### Scenario: Save log settings
- **WHEN** a POST request is sent to `/api/admin/log-settings` with body `[{"module": "email", "level": "INFO"}]`
- **THEN** the email module's verbosity SHALL be updated to INFO
- **THEN** the response SHALL be HTTP 200 with `{"status": "ok"}`

#### Scenario: Invalid level returns 400
- **WHEN** a POST request is sent to `/api/admin/log-settings` with an invalid level (e.g., "TRACE")
- **THEN** the response SHALL be HTTP 400 with an error message

### Requirement: Log settings UI in admin panel

The Logs tab SHALL include a "Settings" section (collapsible or sub-tab) where the admin can:
- View current verbosity per module
- Change the level for any module via a dropdown
- See the default level displayed
- Save changes with a button

#### Scenario: Log settings section visible in Logs tab
- **WHEN** the admin views the Logs tab
- **THEN** a "Settings" section or button SHALL be visible to configure log verbosity

#### Scenario: Change module verbosity from UI
- **WHEN** the admin changes a module's level dropdown to "DEBUG" and clicks Save
- **THEN** a POST request SHALL be sent to `/api/admin/log-settings` with the updated setting
- **THEN** the UI SHALL confirm the save
