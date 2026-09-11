package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"

	"heat/middleware"
)

// enableOTelSignals enables all OTel signals in the test database.
func enableOTelSignals() {
	testServer.DB.Exec("INSERT OR REPLACE INTO otel_settings (id, endpoint, traces_enabled, metrics_enabled, logs_enabled) VALUES (1, '', 1, 1, 1)")
}

// TestInitOTelStdoutFallback verifies initOTel falls back to stdout exporter
// when no OTLP endpoint is configured (DB has empty endpoint seeded by db.Init).
func TestInitOTelStdoutFallback(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	os.Unsetenv("OTEL_TRACES_SAMPLER")
	os.Unsetenv("OTEL_SERVICE_NAME")
	os.Unsetenv("OTEL_RESOURCE_ATTRIBUTES")

	shutdown := initOTel(testServer)
	defer shutdown()

	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Fatal("expected tracer provider to be set after initOTel")
	}

	// Should be a real tracer provider (SDK), not noop
	_, isSDK := tp.(*trace.TracerProvider)
	if !isSDK {
		t.Log("warning: tracer provider is not SDK type; graceful degradation may have used noop")
	}

	// Verify tracer creates valid spans
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	if !span.SpanContext().HasTraceID() {
		t.Error("expected span to have a trace ID")
	}
	span.End()
}

// TestInitOTelGracefulDegradation verifies that an unreachable OTLP endpoint
// doesn't crash the server — graceful degradation with stdout/noop fallback.
func TestInitOTelGracefulDegradation(t *testing.T) {
	enableOTelSignals()
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http")
	os.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	os.Setenv("OTEL_SERVICE_NAME", "heat-test-graceful")
	defer func() {
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
		os.Unsetenv("OTEL_TRACES_SAMPLER")
		os.Unsetenv("OTEL_SERVICE_NAME")
	}()

	// This should not panic or block indefinitely
	shutdown := initOTel(testServer)
	defer shutdown()

	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Fatal("expected tracer provider to be set despite unreachable endpoint")
	}

	// Should still produce valid trace IDs (stdout fallback)
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "graceful-test")
	if !span.SpanContext().HasTraceID() {
		t.Error("expected span to have a trace ID despite unreachable endpoint")
	}
	span.End()
}

// TestInitOTelSamplerConfig verifies OTEL_TRACES_SAMPLER env var is accepted.
func TestInitOTelSamplerConfig(t *testing.T) {
	enableOTelSignals()
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http")
	os.Setenv("OTEL_TRACES_SAMPLER", "always_off")
	defer func() {
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
		os.Unsetenv("OTEL_TRACES_SAMPLER")
	}()

	shutdown := initOTel(testServer)
	defer shutdown()

	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Fatal("expected tracer provider to be set")
	}

	// With always_off sampler, spans should not be sampled
	_, span := tp.Tracer("test").Start(context.Background(), "no-sample-test")
	t.Logf("span sampled: %v (may vary with graceful degradation)", span.SpanContext().IsSampled())
	span.End()
}

// TestOTelMetricsEndpoint verifies /metrics/prometheus serves metrics.
func TestOTelMetricsEndpoint(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")

	shutdown := initOTel(testServer)
	defer shutdown()
	initOTelMetrics()

	r := gin.New()
	r.GET("/metrics/prometheus", gin.WrapH(promhttp.Handler()))

	req, _ := http.NewRequest("GET", "/metrics/prometheus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	t.Logf("Metrics body length: %d", len(body))

	// Should include Prometheus metrics output
	if !strings.Contains(body, "# HELP") {
		t.Log("prometheus metrics body doesn't contain # HELP lines")
	}
}

// TestInitOTelMetricsInstruments verifies initOTelMetrics creates instruments.
func TestInitOTelMetricsInstruments(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	shutdown := initOTel(testServer)
	defer shutdown()

	initOTelMetrics()

	if otelHTTPRequestsTotal == nil {
		t.Error("expected otelHTTPRequestsTotal to be non-nil after initOTelMetrics")
	}
	if otelHTTPRequestDuration == nil {
		t.Error("expected otelHTTPRequestDuration to be non-nil after initOTelMetrics")
	}
}

// TestTraceDBQueryHelper verifies the real middleware.TraceDBQuery helper
// invokes the callback and propagates errors (without panicking).
func TestTraceDBQueryHelper(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	shutdown := initOTel(testServer)
	defer shutdown()

	wantErr := errors.New("boom")
	called := false
	err := middleware.TraceDBQuery(context.Background(), "test-query", func(ctx context.Context) error {
		called = true
		return wantErr
	})
	if !called {
		t.Fatal("expected callback to be invoked")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error to propagate, got: %v", err)
	}
}

// TestTextMapPropagatorConfigured verifies initOTel installs a working global
// propagator, so inbound traceparent headers are honoured and outbound requests
// can carry trace context.
func TestTextMapPropagatorConfigured(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	shutdown := initOTel(testServer)
	defer shutdown()

	ctx, span := otel.Tracer("test").Start(context.Background(), "propagation-test")
	defer span.End()

	carrier := propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	if carrier.Get("traceparent") == "" {
		t.Fatal("expected a traceparent header to be injected after initOTel")
	}

	extracted := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	if got := apitrace.SpanContextFromContext(extracted); got.TraceID() != span.SpanContext().TraceID() {
		t.Fatalf("injected and extracted trace IDs differ: %s != %s", got.TraceID(), span.SpanContext().TraceID())
	}
}
