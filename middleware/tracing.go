package middleware

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// tracer is the OTel tracer used for DB query tracing.
var tracer = otel.Tracer("heat-db")

// TraceDBQuery wraps a database operation in an OTel span with the given operation name.
// The span is created as a child of the provided context, enabling trace propagation
// from the parent request (e.g., from otelgin middleware).
//
// Usage:
//
//	err := middleware.TraceDBQuery(ctx, "GetRacers", func(ctx context.Context) error {
//	    rows, err := db.QueryContext(ctx, "SELECT ...")
//	    return err
//	})
func TraceDBQuery(ctx context.Context, operation string, fn func(context.Context) error) error {
	_, span := tracer.Start(ctx, operation,
		trace.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.system", "sqlite"),
		),
	)
	defer span.End()

	return fn(ctx)
}
