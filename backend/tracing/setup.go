package tracing

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
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"os"
)

// TODO: this should go somwehere else
const pluginVersionEnv = "GF_PLUGIN_VERSION"

func newOpentelemetryTraceProvider(address, pluginID string, customAttributes ...attribute.KeyValue) (*tracesdk.TracerProvider, error) {
	client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(address), otlptracegrpc.WithInsecure())
	exp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, err
	}

	defAttributes := []attribute.KeyValue{semconv.ServiceNameKey.String(pluginID)}
	if pv, ok := os.LookupEnv(pluginVersionEnv); ok {
		defAttributes = append(defAttributes, semconv.ServiceVersionKey.String(pv))
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(defAttributes...),
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

func NewTraceProvider(address, pluginID string, opts Opts) (TracerProvider, error) {
	if address == "" {
		return newNoOpTraceProvider(), nil
	}
	return newOpentelemetryTraceProvider(address, pluginID, opts.CustomAttributes...)
}

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

func InitGlobalTraceProvider(tp TracerProvider, propagator propagation.TextMapPropagator) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
}
