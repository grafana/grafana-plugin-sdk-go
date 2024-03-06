package tracing

import (
	"context"
	"go.opentelemetry.io/otel/codes"
	"runtime"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Opts contains settings for the trace provider and tracer setup that can be configured by the plugin developer.
type Opts struct {
	// CustomAttributes contains custom key value attributes used for the default OpenTelemetry trace provider.
	CustomAttributes []attribute.KeyValue
}

// defaultTracerName is the name for the default tracer that is set up if InitDefaultTracer is never called.
const defaultTracerName = "github.com/grafana/grafana-plugin-sdk-go"

var (
	defaultTracer         trace.Tracer
	defaultTracerInitOnce sync.Once
)

// DefaultTracer returns the default tracer that has been set with InitDefaultTracer.
// If InitDefaultTracer has never been called, the returned default tracer is an OTEL tracer
// with its name set to a generic name (`defaultTracerName`)
func DefaultTracer() trace.Tracer {
	defaultTracerInitOnce.Do(func() {
		// Use a non-nil default tracer if it's not set, for the first call.
		if defaultTracer == nil {
			defaultTracer = &contextualTracer{tracer: otel.Tracer(defaultTracerName)}
		}
	})
	return defaultTracer
}

// InitDefaultTracer sets the default tracer to the specified value.
// This method should only be called once during the plugin's initialization, and it's not safe for concurrent use.
func InitDefaultTracer(tracer trace.Tracer) {
	defaultTracer = &contextualTracer{tracer: tracer}
}

// TraceMethod uses the default tracer to start a span with a name taken from the calling function.
// The Finish return must be called with `defer`, and to make use of the retErr argument the calling
// function must employ it as a named return. Example:
//
//	func TraceMe(ctx context.Context, args...) (result []any, retErr error) {
//	    ctx, span, finish := tracing.TraceMethod(ctx, retErr)
//	    defer finish()
//	}
//
// If the function returns a non-nil error, this will be used to set the error status on the span,
// record the error, and create an error event.
//
// The Finish return may be called with SpanEndOptions. One use this is to conditionally attach a stack
// trace to the span if an error is returned or the calling function panics, like so:
//
// defer finish(trace.WithStackTrace(true))
func TraceMethod(ctx context.Context, retErr error, attributes ...attribute.KeyValue) (context.Context, trace.Span, Finish) {
	// skip 1 to get the name of the caller of TraceMethod
	ctx, span := defaultTracer.Start(ctx, callerName(1))
	span.SetAttributes(attributes...)
	finish := func(spanEndOptions ...trace.SpanEndOption) {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End(spanEndOptions...)
	}
	return ctx, span, finish
}

type Finish func(...trace.SpanEndOption)

// callerName returns the name of the nth calling function up the stack, with the receiver type if called from
// a receiver method. For example:
//
//	func CallMe(maybe bool) string {
//	    return callerName(0)
//	}
//
// will return "CallMe", and
//
//	func (p *phone) Call() string {
//	    return callerName(0)
//	}
//
// will return "(*phone).Call"
func callerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return "(unknown)"
	}
	name := runtime.FuncForPC(pc).Name()
	slash := strings.Index(name, "/")
	if slash < 0 {
		slash = 0
	}
	parts := strings.Split(name[slash:], ".")
	return strings.Join(parts[1:], ".")
}
