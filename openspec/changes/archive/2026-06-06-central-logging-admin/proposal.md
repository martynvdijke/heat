## Why

The application lacks a centralized view for monitoring and debugging. Logs are scattered across `log.Printf` calls in the backend, and there's no way to inspect application activity from the admin panel. Operators need a single place to view logs, adjust verbosity, and configure logging destinations — all without SSH access. Additionally, OpenTelemetry (OTel) dependencies already exist in `go.mod` but are not wired up, leaving a gap in observability.

## What Changes

- **New "Logs" tab** in the admin panel with a real-time log viewer
- **New backend logging service** that captures structured logs from all handlers and services
- **New log settings** endpoint to control log verbosity (default: WARN+) per module
- **OTel integration** — wire existing OTel dependencies to ship logs as OTel signals (logs and traces)
- **Onboard existing settings endpoints** (Email, AI, Notifications, Backup, Analytics) into the central logging view so their activity is visible
- **Log persistence** to a SQLite table for browsing historical logs from the admin panel
- **Log verbosity selector** in the UI (default: warnings and above)

## Capabilities

### New Capabilities
- `admin-log-viewer`: Central log viewer tab in the admin panel, with real-time log streaming, filtering by level/module, and historical browsing
- `log-settings`: Log verbosity configuration per module, with persistence and defaults
- `otel-integration`: Wire existing OTel SDK into the application to export logs and traces

### Modified Capabilities
<!-- No existing spec-level capabilities are changing -->

## Impact

- **Backend**: New `handlers/logs.go`, `handlers/log_settings.go`, new logging service (structured logger replacing ad-hoc `log.Printf` calls)
- **Frontend**: New "Logs" tab in `admin.html`, new JavaScript in `admin.js`, new JS bundle `static/js/logs.js` or inline
- **Database**: New table `app_logs` for log persistence, new table or row `log_settings` for verbosity configuration
- **Dependencies**: Already-present OTel deps (`go.opentelemetry.io/otel`, `otel/sdk`, `otel/stdouttrace`) will be wired in via `telemetry.go`
- **Endpoints**: New API routes under `/api/admin/` for logs and log settings
