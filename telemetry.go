package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	apiometrict "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

	otelHTTPRequestsTotal   apiometrict.Int64Counter
	otelHTTPRequestDuration apiometrict.Float64Histogram
)

// initOTelMetrics creates OTel metric instruments from the global meter provider.
func initOTelMetrics() {
	meter := otel.Meter("heat-http")

	var err error
	otelHTTPRequestsTotal, err = meter.Int64Counter(
		"otel_http_requests_total",
		apiometrict.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		log.Printf("[OTEL] Failed to create OTel request counter: %v", err)
	}

	otelHTTPRequestDuration, err = meter.Float64Histogram(
		"otel_http_request_duration_seconds",
		apiometrict.WithDescription("HTTP request duration in seconds"),
		apiometrict.WithUnit("s"),
	)
	if err != nil {
		log.Printf("[OTEL] Failed to create OTel request duration histogram: %v", err)
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		c.Next()

		status := fmt.Sprintf("%d", c.Writer.Status())
		duration := time.Since(start).Seconds()

		// Prometheus client_golang metrics (existing)
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)

		// OTel metrics (additive, exposed via Prometheus exporter at /metrics/prometheus)
		if otelHTTPRequestsTotal != nil {
			otelHTTPRequestsTotal.Add(c.Request.Context(), 1,
				apiometrict.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("path", path),
					attribute.String("status", status),
				),
			)
		}
		if otelHTTPRequestDuration != nil {
			otelHTTPRequestDuration.Record(c.Request.Context(), duration,
				apiometrict.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("path", path),
				),
			)
		}
	}
}

// initOTel initializes the OpenTelemetry SDK for all three pillars (traces, metrics, logs).
// It reads OTel settings from standard OTEL_* env vars, falling back to database settings.
// Returns a shutdown function to flush and close all providers.
func initOTel(server *app.Server) func() {
	// Prevent OTel export failures from recursing back into the log pipeline:
	// slog.SetDefault redirects the standard library logger into OTel logs, so
	// the default error handler (which uses log.Print) would loop on failures.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		fmt.Fprintf(os.Stderr, "[OTEL] %v\n", err)
	}))

	// Set the global propagator so inbound W3C traceparent headers are
	// extracted (otelgin) and outbound requests can be correlated. Without
	// this the global default is a no-op and distributed traces are dropped.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Detect endpoint from env var first, then DB settings
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		var dbEndpoint string
		err := server.DB.QueryRow("SELECT COALESCE(endpoint, '') FROM otel_settings WHERE id = 1").Scan(&dbEndpoint)
		if err == nil && dbEndpoint != "" {
			endpoint = dbEndpoint
		}
	}

	// Determine which signals are enabled (from DB, default all enabled with endpoint)
	var tracesEnabled, metricsEnabled, logsEnabled int
	err := server.DB.QueryRow(
		"SELECT COALESCE(traces_enabled, 1), COALESCE(metrics_enabled, 1), COALESCE(logs_enabled, 1) FROM otel_settings WHERE id = 1",
	).Scan(&tracesEnabled, &metricsEnabled, &logsEnabled)
	if err != nil {
		tracesEnabled = 1
		metricsEnabled = 1
		logsEnabled = 1
	}

	if endpoint == "" {
		log.Printf("[OTEL] No OTLP endpoint configured, using local exporters")
		return initLocalOTel(server.CurrentVersion, tracesEnabled == 1, metricsEnabled == 1, logsEnabled == 1)
	}

	// Detect OTLP protocol (default: grpc)
	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	if protocol == "" {
		protocol = "grpc"
	}

	// Build resource from env (OTEL_RESOURCE_ATTRIBUTES, OTEL_SERVICE_NAME) and defaults
	mergedRes := buildResource(server.CurrentVersion)

	var shutdownFuncs []func()

	// --- Traces ---
	if tracesEnabled == 1 {
		var traceExporter sdktrace.SpanExporter
		var traceErr error

		switch protocol {
		case "grpc":
			traceExporter, traceErr = otlptracegrpc.New(
				context.Background(),
				otlptracegrpc.WithEndpointURL(endpoint),
				otlptracegrpc.WithTimeout(10*time.Second),
			)
		default:
			traceExporter, traceErr = otlptracehttp.New(
				context.Background(),
				otlptracehttp.WithEndpointURL(endpoint),
				otlptracehttp.WithTimeout(10*time.Second),
			)
		}

		if traceErr != nil {
			log.Printf("[OTEL] Failed to create trace exporter: %v, traces disabled", traceErr)
		} else {
			// Configure sampler from env vars (OTEL_TRACES_SAMPLER, OTEL_TRACES_SAMPLER_ARG).
			// Default follows the OTel spec: parentbased_always_on.
			sampler := sdktrace.ParentBased(sdktrace.AlwaysSample())
			switch os.Getenv("OTEL_TRACES_SAMPLER") {
			case "always_on":
				sampler = sdktrace.AlwaysSample()
			case "always_off":
				sampler = sdktrace.NeverSample()
			case "traceidratio":
				arg := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
				if arg != "" {
					var ratio float64
					if _, err := fmt.Sscanf(arg, "%f", &ratio); err == nil {
						sampler = sdktrace.TraceIDRatioBased(ratio)
					}
				}
			case "parentbased_always_on":
				sampler = sdktrace.ParentBased(sdktrace.AlwaysSample())
			case "parentbased_always_off":
				sampler = sdktrace.ParentBased(sdktrace.NeverSample())
			case "parentbased_traceidratio":
				arg := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
				if arg != "" {
					var ratio float64
					if _, err := fmt.Sscanf(arg, "%f", &ratio); err == nil {
						sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
					}
				}
			}

			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(traceExporter,
					sdktrace.WithBatchTimeout(5*time.Second),
				),
				sdktrace.WithResource(mergedRes),
				sdktrace.WithSampler(sampler),
			)

			otel.SetTracerProvider(tp)
			shutdownFuncs = append(shutdownFuncs, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tp.Shutdown(ctx); err != nil {
					log.Printf("[OTEL] Error shutting down tracer provider: %v", err)
				}
			})
			log.Printf("[OTEL] Trace exporter initialized (protocol=%s, sampler=%s)", protocol, os.Getenv("OTEL_TRACES_SAMPLER"))
		}
	} else {
		log.Printf("[OTEL] Traces disabled via DB settings")
	}

	// --- Metrics ---
	if metricsEnabled == 1 {
		var metricReaders []metric.Reader

		// OTLP periodic reader
		var metricExporter metric.Exporter
		var metricErr error

		switch protocol {
		case "grpc":
			metricExporter, metricErr = otlpmetricgrpc.New(
				context.Background(),
				otlpmetricgrpc.WithEndpointURL(endpoint),
				otlpmetricgrpc.WithTimeout(10*time.Second),
			)
		default:
			metricExporter, metricErr = otlpmetrichttp.New(
				context.Background(),
				otlpmetrichttp.WithEndpointURL(endpoint),
				otlpmetrichttp.WithTimeout(10*time.Second),
			)
		}

		if metricErr != nil {
			log.Printf("[OTEL] Failed to create OTLP metric exporter: %v", metricErr)
		} else {
			metricReaders = append(metricReaders,
				metric.NewPeriodicReader(metricExporter, metric.WithInterval(30*time.Second)),
			)
		}

		// OTel Prometheus exporter (additive pull-based reader)
		promExporter, promErr := otelprometheus.New(
			otelprometheus.WithRegisterer(prometheus.DefaultRegisterer),
		)
		if promErr != nil {
			log.Printf("[OTEL] Failed to create Prometheus exporter: %v", promErr)
		} else {
			metricReaders = append(metricReaders, promExporter)
		}

		if len(metricReaders) > 0 {
			var mpOpts []metric.Option
			for _, r := range metricReaders {
				mpOpts = append(mpOpts, metric.WithReader(r))
			}
			mpOpts = append(mpOpts, metric.WithResource(mergedRes))
			mp := metric.NewMeterProvider(mpOpts...)

			otel.SetMeterProvider(mp)
			shutdownFuncs = append(shutdownFuncs, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := mp.Shutdown(ctx); err != nil {
					log.Printf("[OTEL] Error shutting down meter provider: %v", err)
				}
			})
			log.Printf("[OTEL] Metric exporters initialized (protocol=%s, prometheus=true)", protocol)
		}
	} else {
		log.Printf("[OTEL] Metrics disabled via DB settings")
	}

	// --- Logs ---
	if logsEnabled == 1 {
		var logExporter sdklog.Exporter
		var logErr error

		switch protocol {
		case "grpc":
			logExporter, logErr = otlploggrpc.New(
				context.Background(),
				otlploggrpc.WithEndpointURL(endpoint),
				otlploggrpc.WithTimeout(10*time.Second),
			)
		default:
			logExporter, logErr = otlploghttp.New(
				context.Background(),
				otlploghttp.WithEndpointURL(endpoint),
				otlploghttp.WithTimeout(10*time.Second),
			)
		}

		if logErr != nil {
			log.Printf("[OTEL] Failed to create log exporter: %v, logs disabled", logErr)
		} else {
			lp := sdklog.NewLoggerProvider(
				sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
				sdklog.WithResource(mergedRes),
			)

			global.SetLoggerProvider(lp)

			// Wire the OTel slog bridge so slog records flow through OTel logs with trace context
			setupSlogBridge(lp)
			log.Printf("[OTEL] slog bridge initialized for log-to-trace correlation")

			shutdownFuncs = append(shutdownFuncs, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := lp.Shutdown(ctx); err != nil {
					log.Printf("[OTEL] Error shutting down logger provider: %v", err)
				}
			})
			log.Printf("[OTEL] Log exporter initialized (protocol=%s)", protocol)
		}
	} else {
		log.Printf("[OTEL] Logs disabled via DB settings")
	}

	log.Printf("[OTEL] OpenTelemetry initialized with OTLP endpoint: %s (traces=%v metrics=%v logs=%v)",
		endpoint, tracesEnabled == 1, metricsEnabled == 1, logsEnabled == 1)

	return func() {
		// Shutdown in reverse order (logs first, then metrics, then traces)
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownFuncs[i]()
		}
	}
}

// initLocalOTel initializes the enabled OTel signals with local exporters when
// no OTLP collector is configured: traces to stdout, metrics to the Prometheus
// registry served at /metrics/prometheus, and logs to stdout. This keeps all
// three pillars functional for local development and single-node deployments.
func initLocalOTel(version string, tracesEnabled, metricsEnabled, logsEnabled bool) func() {
	res := buildResource(version)

	var shutdownFuncs []func()

	// --- Traces ---
	if tracesEnabled {
		exporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			log.Printf("[OTEL] Failed to create stdout trace exporter: %v (traces disabled)", err)
		} else {
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(res),
			)
			otel.SetTracerProvider(tp)
			shutdownFuncs = append(shutdownFuncs, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tp.Shutdown(ctx); err != nil {
					log.Printf("[OTEL] Error shutting down tracer provider: %v", err)
				}
			})
			log.Printf("[OTEL] Local trace exporter initialized (stdout)")
		}
	}

	// --- Metrics ---
	if metricsEnabled {
		promExporter, err := otelprometheus.New(
			otelprometheus.WithRegisterer(prometheus.DefaultRegisterer),
		)
		if err != nil {
			log.Printf("[OTEL] Failed to create Prometheus exporter: %v (metrics disabled)", err)
		} else {
			mp := metric.NewMeterProvider(
				metric.WithReader(promExporter),
				metric.WithResource(res),
			)
			otel.SetMeterProvider(mp)
			shutdownFuncs = append(shutdownFuncs, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := mp.Shutdown(ctx); err != nil {
					log.Printf("[OTEL] Error shutting down meter provider: %v", err)
				}
			})
			log.Printf("[OTEL] Local metric exporter initialized (prometheus)")
		}
	}

	// --- Logs ---
	if logsEnabled {
		exporter, err := stdoutlog.New(
			stdoutlog.WithPrettyPrint(),
		)
		if err != nil {
			log.Printf("[OTEL] Failed to create stdout log exporter: %v (logs disabled)", err)
		} else {
			lp := sdklog.NewLoggerProvider(
				sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
				sdklog.WithResource(res),
			)
			global.SetLoggerProvider(lp)
			setupSlogBridge(lp)
			shutdownFuncs = append(shutdownFuncs, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := lp.Shutdown(ctx); err != nil {
					log.Printf("[OTEL] Error shutting down logger provider: %v", err)
				}
			})
			log.Printf("[OTEL] Local log exporter initialized (stdout)")
		}
	}

	log.Printf("[OTEL] OpenTelemetry initialized with local exporters (traces=%v metrics=%v logs=%v)",
		tracesEnabled, metricsEnabled, logsEnabled)

	return func() {
		// Shutdown in reverse order (logs first, then metrics, then traces)
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownFuncs[i]()
		}
	}
}

// buildResource builds the OTel resource from code defaults merged with
// OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES from the environment.
func buildResource(version string) *resource.Resource {
	envRes, _ := resource.New(context.Background(),
		resource.WithFromEnv(),
	)
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("heat"),
		attribute.String("service.version", version),
	)
	// Merge(a, b) lets b win, so env-derived values take precedence.
	merged, _ := resource.Merge(res, envRes)
	return merged
}

// setupSlogBridge routes slog records to both the OTel log pipeline and stderr.
// The stderr handler keeps local console output intact when OTel logs are on.
func setupSlogBridge(lp *sdklog.LoggerProvider) {
	handlers := []slog.Handler{
		slog.NewTextHandler(os.Stderr, nil),
		otelslog.NewHandler("heat", otelslog.WithLoggerProvider(lp)),
	}
	slog.SetDefault(slog.New(fanoutHandler{handlers: handlers}))
}

// fanoutHandler forwards each record to every child handler that accepts it.
// It is used to keep console logging while also exporting to OTel.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (f fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range f.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (f fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return fanoutHandler{handlers: handlers}
}

func (f fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return fanoutHandler{handlers: handlers}
}
