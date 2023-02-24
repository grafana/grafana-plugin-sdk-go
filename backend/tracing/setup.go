package tracing

import (
	"context"
	jaegerpropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func newOpentelemetryTraceProvider(address, pluginID string) (*tracesdk.TracerProvider, error) {
	client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint(address), otlptracegrpc.WithInsecure())
	exp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(pluginID),
			// TODO: version
			semconv.ServiceVersionKey.String("na"),
		),
		// TODO: custom attributes
		// resource.WithAttributes(ots.customAttribs...),
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

func NewTraceProvider(address, pluginID string) (TracerProvider, error) {
	if address == "" {
		return newNoOpTraceProvider(), nil
	}
	return newOpentelemetryTraceProvider(address, pluginID)
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
