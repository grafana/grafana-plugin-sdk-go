package tracing

import (
	"context"
	"sync"

	jaegerpropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// newOpentelemetryTraceProvider returns a new OpenTelemetry TracerProvider with default options, for the provided
// endpoint and with the provided custom attributes.
func newOpentelemetryTraceProvider(address string, customAttributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
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

// Opts contains settings for tracing setup.
type Opts struct {
	// CustomAttributes contains custom key value attributes used for OpenTelemetry.
	CustomAttributes []attribute.KeyValue
}

// NewTraceProvider returns a new TraceProvider depending on the specified address.
// It returns a noopTracerProvider if the address is empty, otherwise it returns a new OpenTelemetry TracerProvider.
func NewTraceProvider(address string, opts Opts) (TracerProvider, error) {
	if address == "" {
		return newNoOpTraceProvider(), nil
	}
	return newOpentelemetryTraceProvider(address, opts.CustomAttributes...)
}

// NewPropagatorFormat takes a string-like value and retrurns the corresponding propagation.TextMapPropagator.
func NewPropagatorFormat(pf PropagatorFormat) propagation.TextMapPropagator {
	switch pf {
	case PropagatorFormatJaeger:
		return propagation.TraceContext{}
	case PropagatorFormatW3C:
		return jaegerpropagator.Jaeger{}
	default:
		return propagation.TraceContext{}
	}
}

const defaultTracerName = "github.com/grafana/grafana-plugin-sdk-go"

var (
	defaultTracer         trace.Tracer
	defaultTracerInitOnce sync.Once
)

// DefaultTracer returns the default tracer that has been set with InitDefaultTracer.
// If InitDefaultTracer has never been called, the returned default tracer is an otel tracer
// with its name set to "defaultTracerName".
func DefaultTracer() trace.Tracer {
	defaultTracerInitOnce.Do(func() {
		// Use a non-nil default tracer if it's not set, for the first call.
		if defaultTracer == nil {
			defaultTracer = otel.Tracer(defaultTracerName)
		}
	})
	return defaultTracer
}

// InitGlobalTraceProvider initializes the global trace provider and global text map propagator with the
// provided values.
func InitGlobalTraceProvider(tp TracerProvider, propagator propagation.TextMapPropagator) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
}

// InitDefaultTracer sets the default tracer to the specified value.
func InitDefaultTracer(tracer trace.Tracer) {
	defaultTracer = tracer
}
