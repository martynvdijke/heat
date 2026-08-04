## Context

heat is a Go 1.26 application using the **Gin** router and **SQLite** via the **Ent ORM**. It currently has a partial OTel setup in `telemetry.go`: a stdout-only trace exporter (`stdouttrace`) with an OTLP HTTP fallback (`otlptracehttp`), and Prometheus `client_golang` metrics middleware exposed at `/metrics/prometheus`. The service name is set to "heat" via `semconv.ServiceNameKey`. There is no OTel metrics SDK, no `otelgin` middleware, no OTel logs, and no DB query tracing. The existing Prometheus `/metrics/prometheus` endpoint must continue working.

The `telemetry.go` file exports `initTelemetry()` (called from `main.go`) and a metrics middleware. The OTel SDK is at v1.44.0 (`otel`, `sdk`, `trace`). The logs SDK is not yet present.

## Goals / Non-Goals

**Goals:**
- Pluggable OTLP exporters (gRPC and HTTP/protobuf) for traces, metrics, and logs
- Gin request tracing via `otelgin` middleware — automatic spans per request
- OTel-native HTTP metrics (request count, duration) exposed alongside existing Prometheus metrics via the OTel Prometheus exporter
- DB query tracing — wrap SQLite/Ent queries with OTel spans
- OTel logs with OTLP export and slog bridge for log-to-trace correlation
- Standard OTel env var support: `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER`, `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_SERVICE_NAME`
- Graceful degradation: no OTel config → noop, partial failure → warn + fallback
- Unit tests for telemetry init and integration test for exporter configuration
- All existing tests pass, CI stays green

**Non-Goals:**
- Not replacing the existing Prometheus `client_golang` metrics — OTel metrics are additive
- Not instrumenting every individual handler (Gin middleware covers the request lifecycle; DB tracing covers queries)
- Not adding OTel auto-instrumentation agents or sidecars
- Not changing the Dockerfile — OTel config is env-var driven

## Decisions

**Decision 1: OTLP gRPC as primary exporter, HTTP/protobuf as secondary**

Both `otlptracegrpc`/`otlpmetricgrpc`/`otlploggrpc` and `otlptracehttp`/`otlpmetrichttp`/`otlploghttp` will be supported. The protocol is selected via `OTEL_EXPORTER_OTLP_PROTOCOL` (default: `grpc`).

Rationale: gRPC is the default OTel protocol and the most efficient for high-throughput. HTTP/protobuf is useful when gRPC is blocked. Supporting both adds minimal binary cost.

Alternative considered: Only HTTP (since `otlptracehttp` is already present). Rejected: gRPC primary aligns with the shared OTel convention across all projects.

**Decision 2: otelgin middleware for request tracing, not manual span creation**

Use `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` middleware.

Rationale: otelgin automatically creates spans with HTTP semantic convention attributes, handles trace context propagation via `traceparent` headers, and sets span status on errors. Writing a custom middleware would duplicate this logic and drift from the spec.

Trade-off: Adds a dependency on `contrib/instrumentation`. The contrib module is maintained by the OTel project and has compatibility guarantees.

**Decision 3: OTel Prometheus exporter for metrics bridge, not dual-instrumentation**

Use `go.opentelemetry.io/otel/exporters/prometheus` to expose OTel metric instruments at the existing `/metrics/prometheus` endpoint alongside the current Prometheus `client_golang` metrics. New OTel metric instruments will be created for HTTP request count and duration.

Rationale: Dual-instrumentation is error-prone and drift-prone. The OTel Prometheus exporter converts OTel metrics to Prometheus text format automatically, so we instrument once with OTel and both sources are served from the same endpoint.

Alternative considered: Replace Prometheus client_golang entirely. Rejected: existing metrics use custom label patterns and alerting rules that would need migration. Additive approach is safer.

**Decision 4: DB query tracing via helper function wrapper**

Create a `middleware/tracing.go` file with a helper function `TraceDBQuery(ctx, operation, dbFunc)` that wraps a SQLite query in an OTel span.

Rationale: The codebase uses Ent ORM which generates query code. A wrapper function allows per-query opt-in without touching every generated call site at once. Key queries (race results, season standings, settings) will be wrapped first.

Trade-off: Not automatic. Developers must remember to use `TraceDBQuery`. Mitigation: centralize in the `db` or `ent` interaction layer where possible.

**Decision 5: Config via standard OTel env vars only, not app config**

The app will NOT read OTEL_* vars from its own config or .env file. It relies on the Go OTel SDK's automatic env var detection.

Rationale: The OTel SDK already reads all standard env vars via `otlpconfig` and `sdk/sdktrace`. Duplicating this is unnecessary and risks drift from the spec.

**Decision 6: File structure — rewrite telemetry.go, add middleware/tracing.go**

- `telemetry.go`: rewritten with pluggable exporter selection for all three pillars, OTel SDK init, sampler config, resource detection
- `middleware/tracing.go`: DB query tracing helper
- `main.go`: add `otelgin.Middleware()` to the Gin router, update `initTelemetry()` call and shutdown

Rationale: telemetry.go is the natural home for OTel initialization. A separate middleware file keeps concerns clean.

**Decision 7: OTel logs with slog bridge**

Add OTel logs SDK (`otel/log v0.20.0`, `sdk/log v0.20.0`) with OTLP log export and the OTel slog bridge to route slog records through the OTel logs SDK with automatic trace context injection.

Rationale: heat likely uses `log/slog` for structured logging. Bridging slog to OTel logs provides log-to-trace correlation without changing every log call site.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| OTLP exporter connection blocks startup | Move exporter connection to background goroutine with timeout; server starts with stdout fallback immediately |
| OTel Prometheus exporter duplicates existing `/metrics/prometheus` output | Namespace OTel metrics with `otel_` prefix to avoid collision; both work independently |
| `otelgin` middleware version compatibility | Pin to same minor version as OTel SDK (v1.44.x / contrib v0.69.x) using go.mod |
| DB query tracing adds overhead to every query | No overhead when no exporter is registered (OTel no-op span is cheap); sampling reduces overhead in production |
| Breaking existing `/metrics/prometheus` scrapers | Additive only — existing Prometheus metrics are untouched |
| Logs SDK is still v0.20.0 (unstable) | Pin version explicitly; API may change in future but v0.20.0 is stable enough for production use |

## Open Questions

- Should we add a health check endpoint for the OTel exporter? — Deferred; not needed until multi-collector setups
- Should OTel metrics replace the Prometheus metrics entirely in a future change? — Out of scope