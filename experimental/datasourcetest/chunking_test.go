package datasourcetest

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/internal/testutil"
)

// TestQueryDataAndChunkedResponsesAreTheSame verifies that the QueryData and QueryChunkedData methods return the
// same responses when called with the same queries.
func TestQueryDataAndChunkedResponsesAreTheSame(t *testing.T) {
	tpQueryData := newTestPlugin(t, &pluginQueryData{})
	defer func() {
		err := tpQueryData.Shutdown()
		if err != nil {
			t.Error("failed to shutdown plugin", err)
		}
	}()

	tpQueryChunked := newTestPlugin(t, &pluginQueryChunked{})
	defer func() {
		err := tpQueryChunked.Shutdown()
		if err != nil {
			t.Error("failed to shutdown plugin", err)
		}
	}()

	ctx := context.Background()

	// Maximum number of queries to test simultaneously
	const maxDataQueries = 3

	// Number of data points to simulate
	tests := []int64{1, 100, 1000, 999, 1001, 10000, 10001, 100000, 100002}

	// Test each combination of queries
	for numQueries := 1; numQueries <= maxDataQueries; numQueries++ {
		// Test each combination of data points
		for n := range len(tests) - numQueries {
			dataPoints := tests[n : n+numQueries]
			t.Run(fmt.Sprintf("queries (%d), with %d points", numQueries, dataPoints), func(t *testing.T) {
				// t.Parallel()

				// Create queries array
				queries := make([]backend.DataQuery, numQueries)
				for i := range queries {
					queries[i] = backend.DataQuery{RefID: strconv.Itoa(i + 1), MaxDataPoints: dataPoints[i]}
				}

				// Query data
				resp, err := tpQueryData.Client.QueryData(ctx, &backend.QueryDataRequest{
					PluginContext: backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}},
					Queries:       queries,
				})
				if err != nil {
					t.Error("QueryData failed", err)
					return
				}

				// Query data with chunking enabled
				chunkedResp, err := tpQueryChunked.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{
					PluginContext: backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}},
					Queries:       queries,
				})
				if err != nil {
					t.Error("QueryChunkedData failed", err)
					return
				}

				// Compare responses
				if diff := cmp.Diff(resp, chunkedResp, cmp.AllowUnexported(data.Field{})); diff != "" {
					t.Errorf("QueryData vs QueryChunkedData mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}

// TestChunkingNotImplemented verifies that the QueryChunkedData method returns an Unimplemented gRPC status code error
// when called on a plugin that does not implement QueryChunkedData.
func TestChunkingNotImplemented(t *testing.T) {
	tpQueryData := newTestPlugin(t, &pluginQueryData{})
	defer func() {
		err := tpQueryData.Shutdown()
		if err != nil {
			t.Error("failed to shutdown plugin", err)
		}
	}()

	ctx := context.Background()

	// Query data with chunking enabled
	_, err := tpQueryData.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{
		PluginContext: backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}},
		Queries:       []backend.DataQuery{},
	})

	if err == nil {
		t.Error("QueryChunkedData succeeded unexpectedly")
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Error("failed to get gRPC status from error")
		return
	}

	if st.Code() != codes.Unimplemented {
		t.Errorf("unexpected gRPC status code: %d", st.Code())
	}
}

// newTestPlugin creates a new test plugin instance
func newTestPlugin[T instancemgmt.Instance](t *testing.T, inst T) TestPlugin {
	t.Helper()

	port, err := testutil.GetFreePort()
	require.NoError(t, err)

	addr := "127.0.0.1:" + strconv.Itoa(port)
	t.Log("plugin addr:", addr)

	// Reset prometheus registry to avoid conflicts between tests
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	factory := datasource.InstanceFactoryFunc(func(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		return inst, nil
	})

	tp, err := Manage(factory, ManageOpts{Address: addr})
	require.NoError(t, err)
	return tp
}

// Sample plugin that implements QueryData
type pluginQueryData struct{}

func (p *pluginQueryData) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		response.Responses[q.RefID] = p.query(q)
	}

	return response, nil
}

func makeTestFrame(refID string) *data.Frame {
	field := data.NewField("test", data.Labels{
		"key": "value",
	}, []int64{})
	frame := data.NewFrame("frame"+refID, field)
	frame.RefID = refID
	frame.Meta = &data.FrameMeta{
		Type: data.FrameTypeNumericLong,
		Path: "hello",
	}
	return frame
}

func (p *pluginQueryData) query(q backend.DataQuery) backend.DataResponse {
	frame := makeTestFrame(q.RefID)
	for i := int64(1); i <= q.MaxDataPoints; i++ {
		frame.Fields[0].Append(i)
	}
	frame2 := makeTestFrame(q.RefID)
	frame2.Name = "second empty frame"
	return backend.DataResponse{
		Frames: data.Frames{frame, frame2},
	}
}

// Sample plugin that implements QueryChunkedData
type pluginQueryChunked struct{}

func (p *pluginQueryChunked) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		if err := p.queryChunked(q, w); err != nil {
			return err
		}
	}
	return nil
}

func (p *pluginQueryChunked) queryChunked(q backend.DataQuery, w backend.ChunkedDataWriter) error {
	ctx := context.Background()
	frameID := "x"

	frame := makeTestFrame(q.RefID) // zero length
	if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
		return err
	}
	frame.Extend(1) // just one at a time

	for i := int64(1); i <= q.MaxDataPoints; i++ {
		frame.Fields[0].SetConcrete(0, i)
		if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
			return err
		}
	}

	// Send an additional frame with the same refId
	frame2 := makeTestFrame(q.RefID)
	frame2.Name = "second empty frame"
	if err := w.WriteFrame(ctx, q.RefID, "y", frame2); err != nil {
		return err
	}

	return nil
}
