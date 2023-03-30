package traceprovider

import (
	"context"
	"fmt"
	"strings"

	jaegerpropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

// newNoOpTraceProvider returns a new noopTracerProvider.
func newNoOpTraceProvider() TracerProvider {
	return &noopTracerProvider{TracerProvider: trace.NewNoopTracerProvider()}
}

// newOpentelemetryTraceProvider returns a new OpenTelemetry TracerProvider with default options, for the provided
// endpoint and with the provided custom attributes.
func newOpentelemetryTraceProvider(address string, customAttributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
	// Same as Grafana core
	client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(address), otlptracegrpc.WithInsecure())
	exp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, err
	}
	return initTracerProvider(exp, customAttributes...)
}

func initTracerProvider(exp tracesdk.SpanExporter, customAttributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
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

// NewTraceProvider returns a new TraceProvider depending on the specified address.
// It returns a noopTracerProvider if the address is empty, otherwise it returns a new OpenTelemetry TracerProvider.
func NewTraceProvider(address string, opts tracing.Opts) (TracerProvider, error) {
	if address == "" {
		return newNoOpTraceProvider(), nil
	}
	return newOpentelemetryTraceProvider(address, opts.CustomAttributes...)
}

// NewTextMapPropagator takes a string-like value and returns the corresponding propagation.TextMapPropagator.
func NewTextMapPropagator(pf string) (propagation.TextMapPropagator, error) {
	var propagators []propagation.TextMapPropagator
	for _, propagatorString := range strings.Split(pf, ",") {
		var propagator propagation.TextMapPropagator
		switch PropagatorFormat(propagatorString) {
		case PropagatorFormatW3C:
			propagator = propagation.TraceContext{}
		case PropagatorFormatJaeger:
			propagator = jaegerpropagator.Jaeger{}
		case "":
			continue
		default:
			return nil, fmt.Errorf("unsupported OpenTelemetry propagator: %q", propagator)
		}
		propagators = append(propagators, propagator)
	}
	switch len(propagators) {
	case 0:
		return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}), nil
	case 1:
		return propagators[0], nil
	default:
		return propagation.NewCompositeTextMapPropagator(propagators...), nil
	}
}

// InitGlobalTraceProvider initializes the global trace provider and global text map propagator with the
// provided values. This function edits the global (process-wide) OTEL trace provider, use with care!
func InitGlobalTraceProvider(tp TracerProvider, propagator propagation.TextMapPropagator) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
}

func InitializeForTestsWithPropagatorFormat(propagatorFormat string) (trace.Tracer, error) {
	exp := tracetest.NewInMemoryExporter()
	tp, _ := initTracerProvider(exp)
	otel.SetTracerProvider(tp)

	propagator, err := NewTextMapPropagator(propagatorFormat)
	if err != nil {
		return nil, err
	}
	otel.SetTextMapPropagator(propagator)
	return tp.Tracer("test"), nil
}

func InitializeForTests() (trace.Tracer, error) {
	return InitializeForTestsWithPropagatorFormat("jaeger,w3c")
}
