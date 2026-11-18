package backend

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tenant"
)

type queryLog struct {
	log.Logger
	messages []string
}

func (l *queryLog) FromContext(context.Context) log.Logger { return l }
func (l *queryLog) Debug(msg string, _ ...any)             { l.messages = append(l.messages, msg) }
func (l *queryLog) Error(msg string, _ ...any)             { l.messages = append(l.messages, msg) }
func (l *queryLog) Info(msg string, _ ...any)              { l.messages = append(l.messages, msg) }

func TestChunkedQueryInstrumentation(t *testing.T) {
	downstream := DownstreamError(errors.New("query failed"))
	for _, tc := range []struct {
		name           string
		tenantID       string
		errors         []error
		terminal       error
		status, source string
	}{
		{name: "success", status: "ok", source: "plugin"},
		{name: "partial failure", tenantID: "example", errors: []error{downstream}, status: "error", source: "downstream"},
		{name: "mixed sources", errors: []error{downstream, errors.New("plugin failure")}, status: "error", source: "plugin"},
		{name: "terminal failure", terminal: downstream, status: "error", source: "downstream"},
		{name: "canceled", terminal: context.Canceled, status: "cancelled", source: "downstream"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			spans := tracetest.NewSpanRecorder()
			tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)).Tracer("test")
			logger := &queryLog{Logger: log.NewNullLogger()}
			plugin := Handlers{QueryChunkedDataHandler: QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, w ChunkedDataWriter) error {
				require.Equal(t, tc.tenantID, tenant.IDFromContext(ctx))
				require.Contains(t, log.ContextualAttributesFromContext(ctx), "test-plugin")
				require.Contains(t, log.ContextualAttributesFromContext(ctx), string(EndpointQueryChunkedData))
				for i, err := range tc.errors {
					require.NoError(t, w.WriteError(ctx, []string{"A", "B"}[i], StatusBadRequest, err))
				}
				return tc.terminal
			})}
			chain, err := HandlerFromMiddlewares(plugin, newTenantIDMiddleware(), newContextualLoggerMiddleware(), NewTracingMiddleware(tracer), NewMetricsMiddleware(registry, "test", false), NewLoggerMiddleware(logger, nil), NewErrorSourceMiddleware())
			require.NoError(t, err)
			ctx := t.Context()
			if tc.tenantID != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(tenant.CtxKey, tc.tenantID))
			}
			writes := 0
			sink := NewChunkedDataWriter(DataFrameFormat_JSON, func(c *pluginv2.QueryChunkedDataResponse) error { writes++; return nil })
			err = chain.QueryChunkedData(ctx, &QueryChunkedDataRequest{PluginContext: PluginContext{PluginID: "test-plugin"}}, sink)
			require.ErrorIs(t, err, tc.terminal)
			require.Equal(t, len(tc.errors), writes)
			metrics, err := registry.Gather()
			require.NoError(t, err)
			require.Len(t, metrics, 1)
			require.Len(t, metrics[0].Metric, 1)
			metric := metrics[0].Metric[0]
			require.Equal(t, 1.0, metric.GetCounter().GetValue())
			labels := map[string]string{}
			for _, l := range metric.Label {
				labels[l.GetName()] = l.GetValue()
			}
			require.Equal(t, map[string]string{"endpoint": string(EndpointQueryChunkedData), "status": tc.status, "status_source": tc.source}, labels)
			require.Len(t, spans.Ended(), 1)
			span := spans.Ended()[0]
			require.Equal(t, "sdk.queryChunkedData", span.Name())
			require.Contains(t, span.Attributes(), attribute.String("request_status", tc.status))
			require.Contains(t, span.Attributes(), attribute.String("status_source", tc.source))
			expected := []string{"Plugin Request Started"}
			for range tc.errors {
				expected = append(expected, "Partial data response error")
			}
			require.Equal(t, append(expected, "Plugin Request Completed"), logger.messages)
		})
	}
}

func TestChunkedQueryCapture(t *testing.T) {
	failure := errors.New("query failed")
	for _, tc := range []struct {
		name                         string
		capture, activity, collision bool
		queryErr, writeErr           error
		writes                       int
	}{
		{name: "off", activity: true},
		{name: "no traffic", capture: true},
		{name: "captured", capture: true, activity: true, writes: 1},
		{name: "reserved ID in use", capture: true, activity: true, collision: true, writes: 1},
		{name: "query failure", capture: true, activity: true, queryErr: failure, writes: 2},
		{name: "writer failure", capture: true, activity: true, writeErr: errors.New("send failed"), writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const refID = "__har__uid"
			ctx := t.Context()
			plugin := Handlers{QueryChunkedDataHandler: QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, w ChunkedDataWriter) error {
				recorder, ok := querycapture.RecorderFromContext(ctx)
				require.Equal(t, tc.capture, ok)
				if ok && tc.activity {
					recorder.Record(querycapture.Interaction{Kind: querycapture.KindSQLQuery, Statement: "SELECT 1"})
				}
				if tc.collision {
					return w.WriteFrame(ctx, refID, "own", data.NewFrame("own"))
				}
				return tc.queryErr
			})}
			chain, err := HandlerFromMiddlewares(plugin, newHARCaptureMiddleware())
			require.NoError(t, err)
			req := &QueryChunkedDataRequest{PluginContext: PluginContext{DataSourceInstanceSettings: &DataSourceInstanceSettings{UID: "uid"}}}
			if tc.capture {
				req.Headers = map[string]string{harCaptureRequestHeader: "true"}
			}
			var chunks []*pluginv2.QueryChunkedDataResponse
			sink := NewChunkedDataWriter(DataFrameFormat_JSON, func(c *pluginv2.QueryChunkedDataResponse) error { chunks = append(chunks, c); return tc.writeErr })
			require.ErrorIs(t, chain.QueryChunkedData(ctx, req, sink), tc.writeErr)
			require.Len(t, chunks, tc.writes)
			if tc.writes == 0 {
				return
			}
			require.Equal(t, refID, chunks[0].RefId)
			if tc.collision {
				require.Equal(t, "own", chunks[0].FrameId)
				return
			}
			var f data.Frame
			require.NoError(t, f.UnmarshalJSON(chunks[0].Frame))
			custom := f.Meta.Custom.(map[string]any)
			require.Contains(t, custom["har"], "SELECT 1")
			if tc.queryErr != nil {
				require.Equal(t, tc.queryErr.Error(), custom["queryError"])
				require.Equal(t, tc.queryErr.Error(), chunks[1].Error)
			}
		})
	}
}

type rawTestSink struct {
	ChunkedDataWriter
	chunks []*pluginv2.QueryChunkedDataResponse
}

func (w *rawTestSink) OnChunk(c *pluginv2.QueryChunkedDataResponse) error {
	w.chunks = append(w.chunks, c)
	return nil
}

func TestChunkedCapturePreservesRawWriter(t *testing.T) {
	original := &pluginv2.QueryChunkedDataResponse{RefId: "A", Frame: []byte("opaque")}
	sink := &rawTestSink{}
	plugin := Handlers{QueryChunkedDataHandler: QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, w ChunkedDataWriter) error {
		recorder, ok := querycapture.RecorderFromContext(ctx)
		require.True(t, ok)
		recorder.Record(querycapture.Interaction{Kind: querycapture.KindSQLQuery, Statement: "SELECT 1"})
		raw, ok := w.(rawChunkSink)
		require.True(t, ok)
		return raw.OnChunk(original)
	})}
	chain, err := HandlerFromMiddlewares(plugin, newHARCaptureMiddleware(), NewErrorSourceMiddleware())
	require.NoError(t, err)
	req := &QueryChunkedDataRequest{Headers: map[string]string{harCaptureRequestHeader: "true"}, Format: DataFrameFormat_JSON}
	require.NoError(t, chain.QueryChunkedData(t.Context(), req, sink))
	require.Len(t, sink.chunks, 2)
	require.Same(t, original, sink.chunks[0])
	require.Equal(t, harResponseRefIDPrefix, sink.chunks[1].RefId)
	require.Contains(t, string(sink.chunks[1].Frame), "SELECT 1")
}

func TestChunkedQueryObserverConcurrentWrites(t *testing.T) {
	const queries = 16
	var written []*pluginv2.QueryChunkedDataResponse
	sink := NewChunkedDataWriter(DataFrameFormat_JSON, func(c *pluginv2.QueryChunkedDataResponse) error {
		written = append(written, c)
		return nil
	})
	observer, writer := observeChunkedQuery(sink, "")
	require.NotImplements(t, (*rawChunkSink)(nil), writer)
	var workers sync.WaitGroup
	for i := range queries {
		workers.Go(func() {
			refID := strconv.Itoa(i)
			err := writer.WriteError(t.Context(), refID, StatusBadRequest, DownstreamError(errors.New("first")))
			if err == nil {
				err = writer.WriteError(t.Context(), refID, StatusInternal, PluginError(errors.New("last")))
			}
			if err != nil {
				t.Errorf("write %s: %v", refID, err)
			}
		})
	}
	workers.Wait()
	require.Len(t, written, 2*queries)
	require.Len(t, observer.result.Responses, queries)
	for _, response := range observer.result.Responses {
		require.EqualError(t, response.Error, "last")
		require.Equal(t, ErrorSourcePlugin, response.ErrorSource)
	}
}

func TestRawQueryErrorKeepsCancellationAndMessage(t *testing.T) {
	sink := &rawTestSink{}
	observer, writer := observeChunkedQuery(sink, "")
	chunk := &pluginv2.QueryChunkedDataResponse{RefId: "A", Error: "request: context canceled", ErrorSource: "downstream"}
	require.NoError(t, writer.(rawChunkSink).OnChunk(chunk))
	require.Same(t, chunk, sink.chunks[0])
	require.EqualError(t, observer.result.Responses["A"].Error, chunk.Error)
	require.Equal(t, RequestStatusCancelled, RequestStatusFromQueryDataResponse(&observer.result, nil))
}
