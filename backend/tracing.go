package backend

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
)

const defaultTracerName = "github.com/grafana/grafana-plugin-sdk-go"

var (
	defaultTracer         trace.Tracer
	defaultTracerInitOnce sync.Once
)

// DefaultTracer returns the default tracer that has been set with initDefaultTracer.
// If initDefaultTracer has never been called, the returned default tracer is an otel tracer
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

// initDefaultTracer sets the default tracer to the specified value.
func initDefaultTracer(tracer trace.Tracer) {
	defaultTracer = tracer
}

// initGlobalTraceProvider initializes the global trace provider and global text map propagator with the
// provided values.
func initGlobalTraceProvider(tp tracing.TracerProvider, propagator propagation.TextMapPropagator) {
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
}
