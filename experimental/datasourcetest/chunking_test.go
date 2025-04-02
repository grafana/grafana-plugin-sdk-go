package datasourcetest

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/grafana/grafana-plugin-sdk-go/internal/testutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	tpQueryDataMulti := newTestPlugin(t, &pluginQueryDataMulti{})
	defer func() {
		err := tpQueryDataMulti.Shutdown()
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

	tpQueryChunkedMulti := newTestPlugin(t, &pluginQueryChunkedMulti{})
	defer func() {
		err := tpQueryChunkedMulti.Shutdown()
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

			t.Logf("Testing with %d queries and %v data points", numQueries, dataPoints)

			// Create queries array
			queries := make([]backend.DataQuery, numQueries)
			for i := range queries {
				queries[i] = backend.DataQuery{RefID: strconv.Itoa(i + 1), MaxDataPoints: dataPoints[i]}
			}

			// Query data (single frame)
			resp, err := tpQueryData.Client.QueryData(ctx, &backend.QueryDataRequest{
				PluginContext: backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}},
				Queries:       queries,
			})
			if err != nil {
				t.Error("QueryData failed", err)
				return
			}

			// Query data (multiple frames)
			respMulti, err := tpQueryDataMulti.Client.QueryData(ctx, &backend.QueryDataRequest{
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

			// Query data with chunking enabled (multiple frames)
			chunkedRespMulti, err := tpQueryChunkedMulti.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{
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
			if diff := cmp.Diff(respMulti, chunkedRespMulti, cmp.AllowUnexported(data.Field{})); diff != "" {
				t.Errorf("QueryData vs QueryChunkedData with multiple frames mismatch (-want +got):\n%s", diff)
			}
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
// This simulates a plugin that returns a single frame.
type pluginQueryData struct{}

func (p *pluginQueryData) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		response.Responses[q.RefID] = p.query(q)
	}

	return response, nil
}

func (p *pluginQueryData) query(q backend.DataQuery) backend.DataResponse {
	frame := data.NewFrame("frame", data.NewField("value", nil, []int64{}))
	for i := int64(1); i <= q.MaxDataPoints; i++ {
		frame.AppendRow(i)
	}
	return backend.DataResponse{
		Frames: data.Frames{frame},
	}
}

// Sample plugin that implements QueryChunkedData
// This simulates a plugin that returns a single frame.
type pluginQueryChunked struct{}

func (p *pluginQueryChunked) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		if err := p.queryChunked(ctx, q, w); err != nil {
			return err
		}
	}

	return nil
}

func (p *pluginQueryChunked) queryChunked(ctx context.Context, q backend.DataQuery, w backend.ChunkedDataWriter) error {
	frameID := "1"
	frame := data.NewFrame("frame", data.NewField("value", nil, []int64{}))
	if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
		return err
	}

	const chunkSize = 1000
	count := 0

	for i := int64(1); i <= q.MaxDataPoints; i++ {
		frame.AppendRow(i)

		count++
		if count >= chunkSize {
			if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
				return err
			}
			frame = frame.EmptyCopy()
			count = 0
		}
	}

	// Remaining rows
	if count > 0 {
		if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
			return err
		}
	}

	return nil
}

// Number of frames to simulate multiple frames per response.
const framesPerResponse = 5

// Sample plugin that implements QueryData
// This simulates a plugin that returns multiple frames.
type pluginQueryDataMulti struct{}

func (p *pluginQueryDataMulti) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		response.Responses[q.RefID] = p.query(q)
	}

	return response, nil
}

func (p *pluginQueryDataMulti) query(q backend.DataQuery) backend.DataResponse {
	var frames data.Frames

	for i := 0; i < framesPerResponse; i++ {
		frame := data.NewFrame("frame"+strconv.Itoa(i), data.NewField("value", nil, []int64{}))
		for i := int64(1); i <= q.MaxDataPoints; i++ {
			frame.AppendRow(i)
		}
		frames = append(frames, frame)
	}

	return backend.DataResponse{
		Frames: frames,
	}
}

// Sample plugin that implements QueryChunkedData
// This simulates a plugin that returns multiple frames.
type pluginQueryChunkedMulti struct{}

func (p *pluginQueryChunkedMulti) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		if err := p.queryChunked(ctx, q, w); err != nil {
			return err
		}
	}

	return nil
}

func (p *pluginQueryChunkedMulti) queryChunked(ctx context.Context, q backend.DataQuery, w backend.ChunkedDataWriter) error {
	for i := 0; i < framesPerResponse; i++ {
		if err := p.writeFrameWithNumber(ctx, i, q, w); err != nil {
			return err
		}
	}

	return nil
}

func (p *pluginQueryChunkedMulti) writeFrameWithNumber(ctx context.Context, frameNum int, q backend.DataQuery, w backend.ChunkedDataWriter) error {
	frameID := strconv.Itoa(frameNum)
	frame := data.NewFrame("frame"+frameID, data.NewField("value", nil, []int64{}))
	if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
		return err
	}

	const chunkSize = 1000
	count := 0

	for i := int64(1); i <= q.MaxDataPoints; i++ {
		frame.AppendRow(i)

		count++
		if count >= chunkSize {
			if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
				return err
			}
			frame = frame.EmptyCopy()
			count = 0
		}
	}

	// Remaining rows
	if count > 0 {
		if err := w.WriteFrame(ctx, q.RefID, frameID, frame); err != nil {
			return err
		}
	}

	return nil
}
