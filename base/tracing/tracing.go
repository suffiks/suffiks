package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name)
}

func Get(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}
