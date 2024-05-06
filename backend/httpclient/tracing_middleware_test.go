package httpclient_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tracerprovider"
)

func TestTracingMiddlewareWithDefaultTracerDataRace(t *testing.T) {
	var tracer trace.Tracer

	mw := httpclient.TracingMiddleware(tracer)
	done := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			rt := mw.CreateMiddleware(httpclient.Options{}, nil)
			require.NotNil(t, rt)
			done <- struct{}{}
		}()
	}
	<-done
	<-done
	close(done)
	require.Nil(t, tracer)
}

func TestTracingMiddleware(t *testing.T) {
	t.Run("GET request that returns 200 OK should start and capture span", func(t *testing.T) {
		spanRecorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
		tracer := provider.Tracer("test")

		finalRoundTripper := httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Request: req}, nil
		})

		mw := httpclient.TracingMiddleware(tracer)
		rt := mw.CreateMiddleware(httpclient.Options{
			Labels: map[string]string{
				"l1": "v1",
				"l2": "v2",
			},
		}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(httpclient.MiddlewareName)
		require.True(t, ok)
		require.Equal(t, httpclient.TracingMiddlewareName, middlewareName.MiddlewareName())

		ctx := context.Background()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}

		spans := spanRecorder.Ended()

		require.Len(t, spans, 1)
		span := spans[0]
		require.Equal(t, "HTTP Outgoing Request", span.Name())
		require.False(t, span.EndTime().IsZero())
		require.False(t, span.Status().Code == codes.Error)
		require.Equal(t, codes.Unset, span.Status().Code)
		require.Empty(t, span.Status().Description)
		require.ElementsMatch(t, []attribute.KeyValue{
			attribute.String("l1", "v1"),
			attribute.String("l2", "v2"),
			semconv.HTTPURL("http://test.com/query"),
			semconv.HTTPMethod(http.MethodGet),
			semconv.HTTPStatusCode(http.StatusOK),
		}, span.Attributes())
	})

	t.Run("GET request that returns 400 Bad Request should start and capture span", func(t *testing.T) {
		spanRecorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
		tracer := provider.Tracer("test")

		finalRoundTripper := httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadRequest, Request: req}, nil
		})

		mw := httpclient.TracingMiddleware(tracer)
		rt := mw.CreateMiddleware(httpclient.Options{
			Labels: map[string]string{
				"l1": "v1",
				"l2": "v2",
			},
		}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(httpclient.MiddlewareName)
		require.True(t, ok)
		require.Equal(t, httpclient.TracingMiddlewareName, middlewareName.MiddlewareName())

		ctx := context.Background()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}

		spans := spanRecorder.Ended()

		require.Len(t, spans, 1)
		span := spans[0]
		require.Equal(t, "HTTP Outgoing Request", span.Name())
		require.False(t, span.EndTime().IsZero())
		require.Equal(t, codes.Error, span.Status().Code)
		require.Equal(t, "error with HTTP status code 400", span.Status().Description)
		require.ElementsMatch(t, []attribute.KeyValue{
			attribute.String("l1", "v1"),
			attribute.String("l2", "v2"),
			semconv.HTTPURL("http://test.com/query"),
			semconv.HTTPMethod(http.MethodGet),
			semconv.HTTPStatusCode(http.StatusBadRequest),
		}, span.Attributes())
	})

	t.Run("POST request that returns 200 OK should start and capture span", func(t *testing.T) {
		spanRecorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
		tracer := provider.Tracer("test")

		resContentLength := int64(10)
		finalRoundTripper := httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Request: req, ContentLength: resContentLength}, nil
		})

		mw := httpclient.TracingMiddleware(tracer)
		rt := mw.CreateMiddleware(httpclient.Options{
			Labels: map[string]string{
				"l1": "v1",
				"l2": "v2",
			},
		}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(httpclient.MiddlewareName)
		require.True(t, ok)
		require.Equal(t, httpclient.TracingMiddlewareName, middlewareName.MiddlewareName())

		ctx := context.Background()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://test.com/query", bytes.NewBufferString("{ \"message\": \"ok\"}"))
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}

		spans := spanRecorder.Ended()

		require.Len(t, spans, 1)
		span := spans[0]
		require.Equal(t, "HTTP Outgoing Request", span.Name())
		require.False(t, span.EndTime().IsZero())
		require.Equal(t, codes.Unset, span.Status().Code)
		require.Empty(t, span.Status().Description)
		require.ElementsMatch(t, []attribute.KeyValue{
			attribute.String("l1", "v1"),
			attribute.String("l2", "v2"),
			semconv.HTTPURL("http://test.com/query"),
			semconv.HTTPMethod(http.MethodPost),
			semconv.HTTPStatusCode(http.StatusOK),
			attribute.Int64("http.content_length", resContentLength),
		}, span.Attributes())
	})

	t.Run("propagation", func(t *testing.T) {
		traceExporter := tracetest.NewInMemoryExporter()
		t.Cleanup(func() {
			require.NoError(t, traceExporter.Shutdown(context.Background()))
		})

		t.Run("single", func(t *testing.T) {
			tracer, err := tracerprovider.InitializeForTestsWithPropagatorFormat("w3c")
			require.NoError(t, err)

			ctx, span := tracer.Start(context.Background(), "testspan")
			defer span.End()

			expectedTraceID := trace.SpanContextFromContext(ctx).TraceID()
			require.NotEmpty(t, expectedTraceID)

			mw := httpclient.TracingMiddleware(tracer)
			rt := mw.CreateMiddleware(httpclient.Options{}, httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// Only w3c header should be present
				require.NotEmpty(t, req.Header.Get("Traceparent"))
				require.Empty(t, req.Header.Get("Uber-Trace-Id"))

				// child span should have the same trace ID as the parent span
				ctx, span := tracer.Start(req.Context(), "inner")
				defer span.End()

				require.Equal(t, expectedTraceID, trace.SpanContextFromContext(ctx).TraceID())

				return &http.Response{StatusCode: http.StatusOK, Request: req}, nil
			}))

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
			require.NoError(t, err)
			res, err := rt.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, req)
			if res.Body != nil {
				require.NoError(t, res.Body.Close())
			}
		})

		t.Run("composite", func(t *testing.T) {
			tracer, err := tracerprovider.InitializeForTests()
			require.NoError(t, err)

			ctx, span := tracer.Start(context.Background(), "testspan")
			defer span.End()

			expectedTraceID := trace.SpanContextFromContext(ctx).TraceID()
			require.NotEmpty(t, expectedTraceID)

			mw := httpclient.TracingMiddleware(tracer)
			rt := mw.CreateMiddleware(httpclient.Options{}, httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// both Jaeger and w3c headers should be set
				require.NotEmpty(t, req.Header.Get("Uber-Trace-Id"))
				require.NotEmpty(t, req.Header.Get("Traceparent"))

				// child span should have the same trace ID as the parent span
				ctx, span := tracer.Start(req.Context(), "inner")
				defer span.End()

				require.Equal(t, expectedTraceID, trace.SpanContextFromContext(ctx).TraceID())

				return &http.Response{StatusCode: http.StatusOK, Request: req}, nil
			}))

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
			require.NoError(t, err)
			res, err := rt.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, req)
			if res.Body != nil {
				require.NoError(t, res.Body.Close())
			}
		})
	})
}
