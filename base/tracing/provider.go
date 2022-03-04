package tracing

import (
	"context"
	"runtime/debug"

	"github.com/go-logr/logr"
	suffiksv1 "github.com/suffiks/suffiks/api/v1"
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

const name = "suffiks"

var (
	tracer   otrace.Tracer = otrace.NewNoopTracerProvider().Tracer("noop")
	provider *trace.TracerProvider
)

func Provider(ctx context.Context, log logr.Logger, cfg suffiksv1.TracingConfig) error {
	if !cfg.Enabled() {
		return nil
	}

	dirty := true
	revision := "unknown"

	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range bi.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				dirty = setting.Value == "true"
			}
		}
	}

	opts := []otlptracegrpc.Option{}
	if cfg.OTLPEndpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
	}
	client := otlptracegrpc.NewClient(opts...)
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return err
	}

	cfgAttrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(name),
		semconv.ServiceVersionKey.String(revision),
	}
	for k, v := range cfg.Attributes {
		cfgAttrs = append(cfgAttrs, attribute.String(k, v))
	}

	if dirty {
		cfgAttrs = append(cfgAttrs, attribute.Bool("modified", true))
	}

	provider = trace.NewTracerProvider(
		// Always be sure to batch in production.
		trace.WithBatcher(exp),
		// Record information about this application in an Resource.
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				cfgAttrs...,
			),
		),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetLogger(log)
	otel.SetErrorHandler(&errorHandler{log: log})
	otel.SetTracerProvider(provider)
	tracer = provider.Tracer(name)
	return nil
}

func Shutdown(ctx context.Context) error {
	if provider == nil {
		return nil
	}

	return provider.Shutdown(ctx)
}

type errorHandler struct {
	log logr.Logger
}

func (e *errorHandler) Handle(err error) {
	log.Log.Error(err, "otel error")
}
