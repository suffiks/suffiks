package tracing

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	otrace "go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	tracer   otrace.Tracer = otrace.NewNoopTracerProvider().Tracer("noop")
	provider *trace.TracerProvider
)

func Provider(ctx context.Context, log logr.Logger, name, version, environment, url string) error {
	opts := []otlptracegrpc.Option{}
	if url != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(url))
	}
	client := otlptracegrpc.NewClient(opts...)
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return err
	}

	parts := strings.Split(name, "/")

	provider = trace.NewTracerProvider(
		// Always be sure to batch in production.
		trace.WithBatcher(exp),
		// Record information about this application in an Resource.
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(parts[len(parts)-1]),
			semconv.ServiceVersionKey.String(version),
			attribute.String("environment", environment),
		)),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetLogger(log)
	otel.SetErrorHandler(&errorHandler{log: log})
	otel.SetTracerProvider(provider)
	tracer = provider.Tracer(name)
	return nil
}

func Shutdown(ctx context.Context) error {
	return provider.Shutdown(ctx)
}

type errorHandler struct {
	log logr.Logger
}

func (e *errorHandler) Handle(err error) {
	log.Log.Error(err, "otel error")
}
