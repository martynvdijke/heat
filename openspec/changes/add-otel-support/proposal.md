## Why

heat currently has a partial OTel setup in `telemetry.go`: a stdout-only trace exporter with an OTLP HTTP fallback (`otlptracehttp`), and Prometheus `client_golang` metrics exposed at `/metrics/prometheus`. There is no OTel metrics SDK, no Gin request instrumentation, no OTel logs, and no DB query tracing. In production, the team has limited visibility into request latency and error rates beyond what Prometheus counters provide, and no distributed tracing or structured log correlation.

Adding full OTel support — traces, metrics, and logs — with OTLP export unlocks distributed tracing, richer metrics, and structured log correlation with observability backends (Grafana Tempo/Loki, Jaeger, SigNoz, etc.) without vendor lock-in. All three pillars aligned on a single OTLP endpoint simplifies the collector story.

## What Changes

- **Add OTLP metric exporters** (gRPC and HTTP/protobuf) alongside the existing Prometheus `client_golang` metrics — OTel metrics are additive, not a replacement
- **Add `otelgin` middleware** via `otelgin` to automatically create spans for every HTTP request with method, path, and status code attributes
- **Add OTel metrics** — OTel SDK metric instruments for HTTP request count (`otel_http_requests_total`) and duration (`otel_http_request_duration_seconds`), bridged to Prometheus via the OTel Prometheus exporter
- **Add DB query tracing** — instrument SQLite/Ent queries with OTel spans to capture DB latency in traces
- **Add OTel logs** — OTel logs SDK with OTLP log export (`otlploggrpc`/`otlploghttp`) and slog bridge for log-to-trace correlation
- **Add OTLP gRPC trace exporter** alongside existing HTTP exporter for gRPC primary alignment
- **Add configurable sampling and resource attributes** — support `OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG`, `OTEL_RESOURCE_ATTRIBUTES` env vars
- **Graceful degradation** — if OTel is not configured (no OTLP endpoint), fall back to no-op propagation without crashing
- **Tests** — unit tests for telemetry initialization and middleware, integration test verifying trace/metric/log export configuration

## Capabilities

### New Capabilities
- `otel-telemetry`: OpenTelemetry-based distributed tracing, metrics, and logs with configurable OTLP export, Gin request instrumentation, and DB query tracing

### Modified Capabilities
<!-- No existing capabilities are having their requirements changed -->

## Impact

- `go.mod`: add `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`, `go.opentelemetry.io/otel/exporters/otlp/otlpmetric`, `otlpmetricgrpc`, `otlpmetrichttp`, `go.opentelemetry.io/otel/sdk/metric`, `go.opentelemetry.io/otel/exporters/prometheus`, `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`, `go.opentelemetry.io/otel/log`, `sdk/log`, `otlploggrpc`, `otlploghttp`
- `telemetry.go`: major rewrite with pluggable exporters for all three pillars, configurable sampling, resource detection
- `main.go`: integrate `otelgin` middleware, update telemetry initialization and shutdown
- New file for DB query tracing helper (e.g., `middleware/tracing.go`)
- New env vars: `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_PROTOCOL`, `OTEL_TRACES_SAMPLER`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`
- `docker-compose.yml`: add `OTEL_*` env vars, document collector endpoint
- CI: no pipeline changes needed — OTel is a pure code addition