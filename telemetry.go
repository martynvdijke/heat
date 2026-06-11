package main

import (
	"fmt"
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"heat/app"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		c.Next()

		status := fmt.Sprintf("%d", c.Writer.Status())
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// initOTel initializes the OpenTelemetry SDK.
// It reads OTel settings from the database to configure OTLP exporters.
// Errors are non-fatal — the app continues without OTel if init fails.
func initOTel(server *app.Server) func() {
	// Read OTel settings from database
	var endpoint string
	var tracesEnabled, metricsEnabled, logsEnabled int
	err := server.DB.QueryRow("SELECT COALESCE(endpoint, ''), COALESCE(traces_enabled, 0), COALESCE(metrics_enabled, 0), COALESCE(logs_enabled, 0) FROM otel_settings WHERE id = 1").
		Scan(&endpoint, &tracesEnabled, &metricsEnabled, &logsEnabled)
	if err != nil {
		log.Printf("[OTEL] No OTel settings found, using stdout exporter (error: %v)", err)
		return initStdoutOTel()
	}

	if endpoint == "" || (tracesEnabled == 0 && metricsEnabled == 0) {
		log.Printf("[OTEL] OTel endpoint not configured or all signals disabled, using stdout exporter")
		return initStdoutOTel()
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("heat"),
		attribute.String("service.version", "1.29.3"),
	)

	var tp *trace.TracerProvider

	if tracesEnabled == 1 {
		traceExporter, err := otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpointURL(endpoint),
			otlptracehttp.WithTimeout(10*time.Second),
		)
		if err != nil {
			log.Printf("[OTEL] Failed to create OTLP trace exporter: %v, falling back to stdout", err)
			return initStdoutOTel()
		}

		tp = trace.NewTracerProvider(
			trace.WithBatcher(traceExporter,
				trace.WithBatchTimeout(5*time.Second),
			),
			trace.WithResource(res),
		)

		otel.SetTracerProvider(tp)
		log.Printf("[OTEL] OpenTelemetry initialized with OTLP endpoint: %s (traces=%v metrics=%v logs=%v)",
			endpoint, tracesEnabled == 1, metricsEnabled == 1, logsEnabled == 1)
	} else {
		log.Printf("[OTEL] Traces disabled, using no-op tracer provider")
		tp = trace.NewTracerProvider()
	}

	return func() {
		if tp != nil {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("[OTEL] Error shutting down tracer provider: %v", err)
			}
		}
	}
}

// initStdoutOTel initializes OTel with a stdout trace exporter (fallback).
func initStdoutOTel() func() {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		log.Printf("[OTEL] Failed to create stdout exporter: %v (continuing without OTel)", err)
		return func() {}
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("heat"),
		attribute.String("service.version", "1.29.3"),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	log.Printf("[OTEL] OpenTelemetry initialized with stdout exporter")
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("[OTEL] Error shutting down tracer provider: %v", err)
		}
	}
}
