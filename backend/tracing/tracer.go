package tracing

import (
	"context"
	
	"go.opentelemetry.io/otel/trace"
)

type TracerProvider interface {
	trace.TracerProvider

	Shutdown(ctx context.Context) error
}

type noopTracerProvider struct {
	trace.TracerProvider
}

func newNoOpTraceProvider() TracerProvider {
	return &noopTracerProvider{TracerProvider: trace.NewNoopTracerProvider()}
}

func (noopTracerProvider) Shutdown(ctx context.Context) error {
	return nil
}
