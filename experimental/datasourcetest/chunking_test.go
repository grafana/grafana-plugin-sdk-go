package datasourcetest

import (
	"context"
	"errors"
	"iter"
	"net/http"
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
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
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

			respJSON, err := tpQueryData.Client.QueryData(ctx, &backend.QueryDataRequest{
				PluginContext: backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}},
				Queries:       queries,
				Format:        backend.DataFrameFormat_JSON,
			})
			if err != nil {
				t.Error("QueryData failed", err)
				return
			}
			if diff := cmp.Diff(resp, respJSON, cmp.AllowUnexported(data.Field{})); diff != "" {
				t.Errorf("QueryData vs QueryData (JSON) mismatch (-want +got):\n%s", diff)
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
			chunkedRespArrow, err := tpQueryChunked.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{
				PluginContext: backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}},
				Queries:       queries,
			})
			if err != nil {
				t.Error("QueryChunkedData failed", err)
				return
			}

			// Query data with chunking enabled
			chunkedRespJSON, err := tpQueryChunked.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{
				Format:        backend.DataFrameFormat_JSON,
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
			if diff := cmp.Diff(resp, chunkedRespArrow, cmp.AllowUnexported(data.Field{})); diff != "" {
				t.Errorf("QueryData vs QueryChunkedData (Arrow) mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(resp, chunkedRespJSON, cmp.AllowUnexported(data.Field{})); diff != "" {
				t.Errorf("QueryData vs QueryChunkedData (JSON) mismatch (-want +got):\n%s", diff)
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

// TestChunkingWithoutHandler verifies that the QueryChunkedData method returns an Unimplemented gRPC status code
// error when the ServeOpts of the plugin have no QueryChunkedDataHandler.
func TestChunkingWithoutHandler(t *testing.T) {
	port, err := testutil.GetFreePort()
	require.NoError(t, err)
	addr := "127.0.0.1:" + strconv.Itoa(port)

	// Reset prometheus registry to avoid conflicts between tests
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	s, err := backend.TestStandaloneServe(backend.ServeOpts{QueryDataHandler: &pluginQueryData{}}, addr)
	require.NoError(t, err)
	tp := TestPlugin{Server: newTestPluginServer(s)}
	defer func() {
		if err := tp.Shutdown(); err != nil {
			t.Error("failed to shutdown plugin", err)
		}
	}()
	tp.Client, err = newTestPluginClient(addr)
	require.NoError(t, err)

	ctx := context.Background()
	queries := []backend.DataQuery{{RefID: "A", MaxDataPoints: 1}}

	_, err = tp.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{Queries: queries})
	require.Equal(t, codes.Unimplemented, status.Code(err), err)

	resp, err := tp.Client.QueryData(ctx, &backend.QueryDataRequest{Queries: queries})
	require.NoError(t, err)
	require.Contains(t, resp.Responses, "A")
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

	for i := range framesPerResponse {
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
	for i := range framesPerResponse {
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

// TestAccumulateFrameIDRule verifies that AccumulateArrow and AccumulateJSON apply the frame ID rule of
// QueryChunkedDataResponse to the chunks of the SDK writer.
func TestAccumulateFrameIDRule(t *testing.T) {
	ctx := context.Background()
	errBadQuery := backend.DownstreamError(errors.New("bad query"))
	errPlugin := errors.New("plugin failed")

	frame := func(name string, values ...int64) *data.Frame {
		return data.NewFrame(name, data.NewField("value", nil, values))
	}
	wantFrame := func(refID, name string, values ...int64) *data.Frame {
		f := frame(name, values...)
		f.RefID = refID
		return f
	}

	type result struct {
		frames      data.Frames
		status      backend.Status
		err         string
		errorSource backend.ErrorSource
	}

	tests := []struct {
		name string
		// write sends the chunks in order. rawFrame sends a frame chunk without a frame ID, which the writer rejects.
		write func(w backend.ChunkedDataWriter, rawFrame func(refID string, f *data.Frame)) error
		want  map[string]result
	}{
		{
			name: "one frame in many chunks",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteFrame(ctx, "A", "1", frame("table", 1, 2)),
					w.WriteFrame(ctx, "A", "1", frame("table", 3)),
					w.WriteFrame(ctx, "A", "1", frame("table", 4, 5)),
				)
			},
			want: map[string]result{
				"A": {frames: data.Frames{wantFrame("A", "table", 1, 2, 3, 4, 5)}, status: backend.StatusOK},
			},
		},
		{
			name: "many frames for one refID",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteFrame(ctx, "A", "1", frame("series1", 1)),
					w.WriteFrame(ctx, "A", "2", frame("series2", 2)),
					w.WriteFrame(ctx, "A", "2", frame("series2", 3)),
					w.WriteFrame(ctx, "A", "3", frame("series3", 4)),
				)
			},
			want: map[string]result{
				"A": {frames: data.Frames{
					wantFrame("A", "series1", 1),
					wantFrame("A", "series2", 2, 3),
					wantFrame("A", "series3", 4),
				}, status: backend.StatusOK},
			},
		},
		{
			name: "alternating frame IDs",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteFrame(ctx, "A", "x", frame("x", 1)),
					w.WriteFrame(ctx, "A", "y", frame("y", 10)),
					w.WriteFrame(ctx, "A", "x", frame("x", 2)),
				)
			},
			want: map[string]result{
				"A": {frames: data.Frames{wantFrame("A", "x", 1, 2), wantFrame("A", "y", 10)}, status: backend.StatusOK},
			},
		},
		{
			name: "same frame ID for two refIDs",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteFrame(ctx, "A", "1", frame("a", 1)),
					w.WriteFrame(ctx, "B", "1", frame("b", 2)),
					w.WriteFrame(ctx, "A", "1", frame("a", 3)),
				)
			},
			want: map[string]result{
				"A": {frames: data.Frames{wantFrame("A", "a", 1, 3)}, status: backend.StatusOK},
				"B": {frames: data.Frames{wantFrame("B", "b", 2)}, status: backend.StatusOK},
			},
		},
		{
			name: "frames without a frame ID",
			write: func(_ backend.ChunkedDataWriter, rawFrame func(string, *data.Frame)) error {
				rawFrame("A", frame("first", 1))
				rawFrame("A", frame("second", 2))
				return nil
			},
			want: map[string]result{
				"A": {frames: data.Frames{wantFrame("A", "first", 1), wantFrame("A", "second", 2)}, status: backend.StatusOK},
			},
		},
		{
			name: "error after frames",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteFrame(ctx, "A", "1", frame("table", 1)),
					w.WriteFrame(ctx, "A", "1", frame("table", 2)),
					w.WriteError(ctx, "A", backend.StatusBadRequest, errBadQuery),
				)
			},
			want: map[string]result{
				"A": {
					frames:      data.Frames{wantFrame("A", "table", 1, 2)},
					status:      backend.StatusBadRequest,
					err:         errBadQuery.Error(),
					errorSource: backend.ErrorSourceDownstream,
				},
			},
		},
		{
			name: "only an error",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteError(ctx, "A", backend.StatusUnauthorized, errBadQuery),
				)
			},
			want: map[string]result{
				"A": {status: backend.StatusUnauthorized, err: errBadQuery.Error(), errorSource: backend.ErrorSourceDownstream},
			},
		},
		{
			name: "frame after an error",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteError(ctx, "A", backend.StatusBadRequest, errBadQuery),
					w.WriteFrame(ctx, "A", "1", frame("table", 1)),
				)
			},
			want: map[string]result{
				"A": {
					frames:      data.Frames{wantFrame("A", "table", 1)},
					status:      backend.StatusBadRequest,
					err:         errBadQuery.Error(),
					errorSource: backend.ErrorSourceDownstream,
				},
			},
		},
		{
			name: "error without a status",
			write: func(w backend.ChunkedDataWriter, _ func(string, *data.Frame)) error {
				return errors.Join(
					w.WriteError(ctx, "A", 0, errPlugin),
				)
			},
			want: map[string]result{
				"A": {status: backend.StatusInternal, err: errPlugin.Error(), errorSource: backend.ErrorSourceFromError(errPlugin)},
			},
		},
	}

	formats := []struct {
		name       string
		format     backend.DataFrameFormat
		accumulate func(iter.Seq2[*pluginv2.QueryChunkedDataResponse, error]) (*backend.QueryDataResponse, error)
	}{
		{name: "arrow", format: backend.DataFrameFormat_ARROW, accumulate: AccumulateArrow},
		{name: "json", format: backend.DataFrameFormat_JSON, accumulate: AccumulateJSON},
	}

	for _, tt := range tests {
		for _, f := range formats {
			t.Run(tt.name+"/"+f.name, func(t *testing.T) {
				var chunks []*pluginv2.QueryChunkedDataResponse
				w := backend.NewChunkedDataWriter(f.format, func(chunk *pluginv2.QueryChunkedDataResponse) error {
					chunks = append(chunks, chunk)
					return nil
				})
				require.NoError(t, tt.write(w, func(refID string, fr *data.Frame) {
					chunks = append(chunks, frameChunkWithoutID(t, f.format, refID, fr))
				}))

				got, err := f.accumulate(chunkSeq(chunks))
				require.NoError(t, err)
				require.Len(t, got.Responses, len(tt.want))
				for refID, want := range tt.want {
					rsp, ok := got.Responses[refID]
					require.True(t, ok, "missing refID %s", refID)
					if diff := cmp.Diff(want.frames, rsp.Frames, data.FrameTestCompareOptions()...); diff != "" {
						t.Errorf("frames of refID %s mismatch (-want +got):\n%s", refID, diff)
					}
					require.Equal(t, want.status, rsp.Status)
					require.Equal(t, want.errorSource, rsp.ErrorSource)
					if want.err == "" {
						require.NoError(t, rsp.Error)
					} else {
						require.EqualError(t, rsp.Error, want.err)
					}
				}
			})
		}
	}
}

// TestAccumulateRejectsMismatchedFields verifies that a later chunk of a frame must have the fields of the first chunk.
func TestAccumulateRejectsMismatchedFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		first, later *data.Frame
	}{
		{
			name:  "other field type",
			first: data.NewFrame("table", data.NewField("value", nil, []int64{1})),
			later: data.NewFrame("table", data.NewField("value", nil, []string{"a"})),
		},
		{
			name: "fewer fields",
			first: data.NewFrame("table",
				data.NewField("value", nil, []int64{1}),
				data.NewField("other", nil, []int64{2}),
			),
			later: data.NewFrame("table", data.NewField("value", nil, []int64{3})),
		},
	}

	formats := []struct {
		name       string
		format     backend.DataFrameFormat
		accumulate func(iter.Seq2[*pluginv2.QueryChunkedDataResponse, error]) (*backend.QueryDataResponse, error)
	}{
		{name: "arrow", format: backend.DataFrameFormat_ARROW, accumulate: AccumulateArrow},
		{name: "json", format: backend.DataFrameFormat_JSON, accumulate: AccumulateJSON},
	}

	for _, tt := range tests {
		for _, f := range formats {
			t.Run(tt.name+"/"+f.name, func(t *testing.T) {
				var chunks []*pluginv2.QueryChunkedDataResponse
				w := backend.NewChunkedDataWriter(f.format, func(chunk *pluginv2.QueryChunkedDataResponse) error {
					chunks = append(chunks, chunk)
					return nil
				})
				require.NoError(t, w.WriteFrame(ctx, "A", "1", tt.first))
				require.NoError(t, w.WriteFrame(ctx, "A", "1", tt.later))

				_, err := f.accumulate(chunkSeq(chunks))
				require.Error(t, err)
			})
		}
	}
}

// TestQueryChunkedDataErrorAfterFrames verifies that QueryData and QueryChunkedData return the same response when
// an error follows the frames of a query.
func TestQueryChunkedDataErrorAfterFrames(t *testing.T) {
	tpQueryData := newTestPlugin(t, &pluginQueryDataError{})
	defer func() {
		if err := tpQueryData.Shutdown(); err != nil {
			t.Error("failed to shutdown plugin", err)
		}
	}()

	tpQueryChunked := newTestPlugin(t, &pluginQueryChunkedError{})
	defer func() {
		if err := tpQueryChunked.Shutdown(); err != nil {
			t.Error("failed to shutdown plugin", err)
		}
	}()

	ctx := context.Background()
	pCtx := backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "1"}}
	queries := []backend.DataQuery{{RefID: "A", MaxDataPoints: 3}, {RefID: "B", MaxDataPoints: 1}}

	want, err := tpQueryData.Client.QueryData(ctx, &backend.QueryDataRequest{PluginContext: pCtx, Queries: queries})
	require.NoError(t, err)

	errorText := cmp.Comparer(func(a, b error) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}
		return a.Error() == b.Error()
	})

	for _, format := range []backend.DataFrameFormat{backend.DataFrameFormat_ARROW, backend.DataFrameFormat_JSON} {
		got, err := tpQueryChunked.Client.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{
			PluginContext: pCtx,
			Queries:       queries,
			Format:        format,
		})
		require.NoError(t, err)
		if diff := cmp.Diff(want, got, cmp.AllowUnexported(data.Field{}), errorText); diff != "" {
			t.Errorf("QueryData vs QueryChunkedData (format %d) mismatch (-want +got):\n%s", format, diff)
		}
	}
}

// chunkSeq returns the chunks as the stream that the accumulators read.
func chunkSeq(chunks []*pluginv2.QueryChunkedDataResponse) iter.Seq2[*pluginv2.QueryChunkedDataResponse, error] {
	return func(yield func(*pluginv2.QueryChunkedDataResponse, error) bool) {
		for _, chunk := range chunks {
			if !yield(chunk, nil) {
				return
			}
		}
	}
}

// frameChunkWithoutID encodes a frame chunk without a frame ID. The SDK writer rejects an empty frame ID, but a
// receiver must still accept one from other producers.
func frameChunkWithoutID(t *testing.T, format backend.DataFrameFormat, refID string, f *data.Frame) *pluginv2.QueryChunkedDataResponse {
	t.Helper()
	f.SetRefID(refID)

	var b []byte
	var err error
	if format == backend.DataFrameFormat_JSON {
		b, err = data.FrameToJSON(f, data.IncludeAll)
	} else {
		b, err = f.MarshalArrow()
	}
	require.NoError(t, err)

	return &pluginv2.QueryChunkedDataResponse{
		RefId:  refID,
		Frame:  b,
		Status: http.StatusOK,
		Format: pluginv2.DataFrameFormat(format),
	}
}

var errQueryAfterFrames = backend.DownstreamError(errors.New("query failed after some rows"))

// Sample plugin that implements QueryData
// This simulates a plugin that returns a frame and an error for each query.
type pluginQueryDataError struct{}

func (p *pluginQueryDataError) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		frame := data.NewFrame("frame", data.NewField("value", nil, []int64{}))
		for i := int64(1); i <= q.MaxDataPoints; i++ {
			frame.AppendRow(i)
		}
		response.Responses[q.RefID] = backend.DataResponse{
			Frames:      data.Frames{frame},
			Error:       errQueryAfterFrames,
			Status:      backend.StatusBadRequest,
			ErrorSource: backend.ErrorSourceDownstream,
		}
	}
	return response, nil
}

// Sample plugin that implements QueryChunkedData
// This simulates a plugin that writes one row per chunk, and then an error for each query.
type pluginQueryChunkedError struct{}

func (p *pluginQueryChunkedError) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	for _, q := range req.Queries {
		for i := int64(1); i <= q.MaxDataPoints; i++ {
			frame := data.NewFrame("frame", data.NewField("value", nil, []int64{i}))
			if err := w.WriteFrame(ctx, q.RefID, "1", frame); err != nil {
				return err
			}
		}
		if err := w.WriteError(ctx, q.RefID, backend.StatusBadRequest, errQueryAfterFrames); err != nil {
			return err
		}
	}
	return nil
}
