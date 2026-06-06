## ADDED Requirements

### Requirement: OTel SDK is initialized at application startup

The system SHALL initialize the OpenTelemetry SDK at startup in `main.go` (before Gin starts). It SHALL use the existing `go.opentelemetry.io/otel/sdk` and `go.opentelemetry.io/otel/exporters/stdout/stdouttrace` dependencies already in `go.mod`. Initialization failures SHALL be logged but SHALL NOT prevent the application from starting.

#### Scenario: OTel initialized before Gin
- **WHEN** the application starts
- **THEN** the OTel SDK SHALL be initialized before Gin routes are registered
- **THEN** the stdouttrace exporter SHALL be configured

#### Scenario: OTel init failure is non-fatal
- **WHEN** OTel SDK initialization fails
- **THEN** the error SHALL be logged via `log.Printf`
- **THEN** the application SHALL continue to start

### Requirement: Structured logs produce OTel log records

When the application's structured logger records a log entry, it SHALL also produce an OpenTelemetry log record via the OTel Logger. The log record SHALL include:
- Timestamp from the log entry
- Severity mapped from the log level (DEBUG→SEVERITY_DEBUG, INFO→SEVERITY_INFO, WARN→SEVERITY_WARN, ERROR→SEVERITY_ERROR)
- Body set to the log message
- Module attribute set as a log record attribute

#### Scenario: Log entry produces OTel record
- **WHEN** the application logger records a WARN entry with module "email" and message "SMTP timeout"
- **THEN** an OTel log record SHALL be produced with severity WARN, body "SMTP timeout", and attribute `module=email`

### Requirement: OTel trace context is propagated to logs

Each log record produced by the OTel bridge SHALL include the current span context (trace_id, span_id) if a span is active in the Go context. This enables correlation between HTTP requests and log entries.

#### Scenario: Log with active span includes trace ID
- **WHEN** a handler creates a span and logs within that span context
- **THEN** the resulting log record SHALL include the active trace_id and span_id

#### Scenario: Log without active span has no trace context
- **WHEN** a log call is made outside any span context
- **THEN** the resulting log record SHALL omit trace_id and span_id
