## ADDED Requirements

### Requirement: Admin can test Umami connection from admin panel

The system SHALL provide a "Test Connection" button in the Analytics settings pane that allows the admin to verify the configured Umami instance is reachable and serving the tracking script.

#### Scenario: Successful connection test
- **WHEN** admin clicks "Test Connection" with a valid Umami URL and Website ID configured
- **THEN** the system sends a server-side request to `<umami_url>/script.js`
- **AND** if the server responds with HTTP 200-299 and a non-empty body containing "umami"
- **THEN** the system displays a success toast: "Connection successful — Umami instance is reachable"

#### Scenario: Connection test fails — unreachable URL
- **WHEN** admin clicks "Test Connection" with an Umami URL that is unreachable or returns an error
- **THEN** the system displays an error toast: "Connection failed — unable to reach Umami instance at <url>"

#### Scenario: Connection test with empty or invalid settings
- **WHEN** admin clicks "Test Connection" with an empty URL or Website ID
- **THEN** the system displays an error toast: "Please save your Umami URL and Website ID before testing"

### Requirement: Backend connection test endpoint

The system SHALL expose a `POST /api/umami-settings/test` endpoint that performs the connection test server-side and returns a JSON result.

#### Scenario: Test endpoint success response
- **WHEN** the handler receives a POST to `/api/umami-settings/test`
- **AND** the configured Umami URL is reachable and serves the tracking script
- **THEN** the endpoint returns HTTP 200 with JSON `{"status": "ok", "message": "Umami instance is reachable"}`

#### Scenario: Test endpoint failure response
- **WHEN** the handler receives a POST to `/api/umami-settings/test`
- **AND** the configured Umami URL is unreachable or does not serve the expected script
- **THEN** the endpoint returns HTTP 200 with JSON `{"status": "error", "message": "<error details>"}`

### Requirement: Input validation on Umami URL and Website ID

The system SHALL validate the Umami URL and Website ID format both client-side (before form submission) and server-side (before persisting).

#### Scenario: Server rejects invalid URL format
- **WHEN** admin submits the form with an invalid URL (e.g., not a valid HTTP/HTTPS URL, missing scheme, or contains spaces)
- **THEN** the server returns HTTP 400 with JSON `{"error": "invalid Umami URL"}`
- **AND** the settings are NOT saved

#### Scenario: Server rejects invalid Website ID format
- **WHEN** admin submits the form with a Website ID that is not a valid UUID format
- **THEN** the server returns HTTP 400 with JSON `{"error": "invalid Website ID — must be a valid UUID"}`
- **AND** the settings are NOT saved

#### Scenario: Client-side validation shows inline errors
- **WHEN** admin tries to save with a missing URL or invalid Website ID
- **THEN** the form highlights the invalid field with a red border
- **AND** shows an inline error message below the field
- **AND** does NOT submit the form
