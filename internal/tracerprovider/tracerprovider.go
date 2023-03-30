package tracerprovider

import (
	"context"

	jaegerpropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
)

// TracerProvider provides a tracer that can be used to instrument a plugin with tracing.
type TracerProvider interface {
	trace.TracerProvider

	// Shutdown performs cleanup operations to ensure the trace provider is disposed correctly.
	Shutdown(ctx context.Context) error
}

// noopTracerProvider is a TracerProvider that uses an no-op underlying trace provider.
type noopTracerProvider struct {
	trace.TracerProvider
}

// Shutdown does nothing and always returns nil.
func (noopTracerProvider) Shutdown(_ context.Context) error {
	return nil
}

// newNoOpTracerProvider returns a new noopTracerProvider.
func newNoOpTracerProvider() TracerProvider {
	return &noopTracerProvider{TracerProvider: trace.NewNoopTracerProvider()}
}

// newOpentelemetryTracerProvider returns a new OpenTelemetry TracerProvider with default options, for the provided
// endpoint and with the provided custom attributes.
func newOpentelemetryTracerProvider(address string, customAttributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
	// Same as Grafana core
	client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(address), otlptracegrpc.WithInsecure())
	exp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(customAttributes...),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithSampler(tracesdk.ParentBased(
			tracesdk.AlwaysSample(),
		)),
		tracesdk.WithResource(res),
	)
	return tp, nil
}

// NewTracerProvider returns a new TracerProvider depending on the specified address.
// It returns a noopTracerProvider if the address is empty, otherwise it returns a new OpenTelemetry TracerProvider.
func NewTracerProvider(address string, opts tracing.Opts) (TracerProvider, error) {
	if address == "" {
		return newNoOpTracerProvider(), nil
	}
	return newOpentelemetryTracerProvider(address, opts.CustomAttributes...)
}

// NewTextMapPropagator takes a string-like value and returns the corresponding propagation.TextMapPropagator.
func NewTextMapPropagator(pf PropagatorFormat) propagation.TextMapPropagator {
	switch pf {
	case PropagatorFormatJaeger:
		return propagation.TraceContext{}
	case PropagatorFormatW3C:
		return jaegerpropagator.Jaeger{}
	default:
		return propagation.TraceContext{}
	}
}

// InitGlobalTracerProvider initializes the global trace provider and global text map propagator with the
// provided values. This function edits the global (process-wide) OTEL trace provider, use with care!
func InitGlobalTracerProvider(tp TracerProvider, propagator propagation.TextMapPropagator) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
}
