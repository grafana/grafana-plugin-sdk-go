package httpclient_test

import (
	"bytes"
	"context"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"testing"
)

type mockTraceProvider struct{}

var _ trace.TracerProvider = mockTraceProvider{}

func (p mockTraceProvider) Tracer(string, ...trace.TracerOption) trace.Tracer {
	return &mockTracer{}
}

type mockTracer struct {
	spans []*mockSpan
}

var _ trace.Tracer = &mockTracer{}

func (t *mockTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	config := trace.NewSpanStartConfig(opts...)
	span := &mockSpan{}
	span.SetName(name)
	span.SetAttributes(config.Attributes()...)
	t.spans = append(t.spans, span)
	return trace.ContextWithSpan(ctx, span), span
}

// mockSpan is an implementation of Span that preforms no operations.
type mockSpan struct {
	name  string
	ended bool

	errored bool
	errs    []error

	statusCode    codes.Code
	statusMessage string

	attributes []attribute.KeyValue
	events     []string
}

var _ trace.Span = &mockSpan{}

// checkValid panics if s has already ended, otherwise it does nothing.
// This ensures that ended spans are never edited afterwards.
func (s *mockSpan) checkValid() {
	if s.ended {
		panic("span already ended")
	}
}

func (s *mockSpan) attributesMap() map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value, len(s.attributes))
	for _, attr := range s.attributes {
		m[attr.Key] = attr.Value
	}
	return m
}

func (*mockSpan) SpanContext() trace.SpanContext { return trace.SpanContext{} }

func (*mockSpan) IsRecording() bool { return false }

func (s *mockSpan) SetStatus(code codes.Code, message string) {
	s.checkValid()
	s.statusCode = code
	s.statusMessage = message
}

func (s *mockSpan) SetError(errored bool) {
	s.checkValid()
	s.errored = true
}

func (s *mockSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.checkValid()
	s.attributes = append(s.attributes, kv...)
}

func (s *mockSpan) End(...trace.SpanEndOption) {
	s.checkValid()
	s.ended = true
}

func (s *mockSpan) RecordError(err error, opts ...trace.EventOption) {
	s.checkValid()
	s.errs = append(s.errs, err)
}

func (s *mockSpan) AddEvent(event string, opts ...trace.EventOption) {
	s.checkValid()
	s.events = append(s.events, event)
}

func (s *mockSpan) SetName(name string) {
	s.checkValid()
	s.name = name
}

func (*mockSpan) TracerProvider() trace.TracerProvider { return mockTraceProvider{} }

func TestTracingMiddleware(t *testing.T) {
	t.Run("GET request that returns 200 OK should start and capture span", func(t *testing.T) {
		tracer := &mockTracer{}

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

		require.Len(t, tracer.spans, 1)
		span := tracer.spans[0]
		require.Equal(t, "HTTP Outgoing Request", span.name)
		require.True(t, span.ended)
		require.False(t, span.errored)
		require.Equal(t, codes.Unset, span.statusCode)
		require.Empty(t, span.statusMessage)
		require.Equal(t, map[attribute.Key]attribute.Value{
			"l1":               attribute.StringValue("v1"),
			"l2":               attribute.StringValue("v2"),
			"http.url":         attribute.StringValue("http://test.com/query"),
			"http.method":      attribute.StringValue("GET"),
			"http.status_code": attribute.Int64Value(200),
		}, span.attributesMap())
	})

	t.Run("GET request that returns 400 Bad Request should start and capture span", func(t *testing.T) {
		tracer := &mockTracer{}

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

		require.Len(t, tracer.spans, 1)
		span := tracer.spans[0]
		require.Equal(t, "HTTP Outgoing Request", span.name)
		require.True(t, span.ended)
		require.False(t, span.errored)
		require.Equal(t, codes.Error, span.statusCode)
		require.Equal(t, "error with HTTP status code 400", span.statusMessage)
		require.Equal(t, map[attribute.Key]attribute.Value{
			"l1":               attribute.StringValue("v1"),
			"l2":               attribute.StringValue("v2"),
			"http.url":         attribute.StringValue("http://test.com/query"),
			"http.method":      attribute.StringValue("GET"),
			"http.status_code": attribute.Int64Value(400),
		}, span.attributesMap())
	})

	t.Run("POST request that returns 200 OK should start and capture span", func(t *testing.T) {
		tracer := &mockTracer{}

		finalRoundTripper := httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Request: req, ContentLength: 10}, nil
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

		require.Len(t, tracer.spans, 1)
		span := tracer.spans[0]
		require.Equal(t, "HTTP Outgoing Request", span.name)
		require.True(t, span.ended)
		require.False(t, span.errored)
		require.Equal(t, codes.Unset, span.statusCode)
		require.Empty(t, span.statusMessage)
		attrMap := span.attributesMap()
		_, ok = attrMap["http.content_length"]
		require.True(t, ok, "http.content_length does not exist")
		delete(attrMap, "http.content_length")
		require.Equal(t, map[attribute.Key]attribute.Value{
			"l1":               attribute.StringValue("v1"),
			"l2":               attribute.StringValue("v2"),
			"http.url":         attribute.StringValue("http://test.com/query"),
			"http.method":      attribute.StringValue("POST"),
			"http.status_code": attribute.Int64Value(200),
		}, attrMap)
	})
}
