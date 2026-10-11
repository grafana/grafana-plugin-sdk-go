package backend

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func chunkedClient(t *testing.T, handler QueryChunkedDataHandler) pluginv2.DataClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pluginv2.RegisterDataServer(srv, newDataSDKAdapter(nil, handler))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pluginv2.NewDataClient(conn)
}

func drain(t *testing.T, stream grpc.ServerStreamingClient[pluginv2.QueryChunkedDataResponse]) (int64, string, error) {
	t.Helper()
	var sum int64
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return sum, firstOrEmpty(stream.Trailer().Get(ResponseSizeBytesTrailerKey)), nil
		}
		if err != nil {
			return sum, firstOrEmpty(stream.Trailer().Get(ResponseSizeBytesTrailerKey)), err
		}
		sum += int64(len(chunk.Frame))
	}
}

func firstOrEmpty(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func TestChunkedDataSetsResponseSizeTrailer(t *testing.T) {
	for _, format := range []pluginv2.DataFrameFormat{pluginv2.DataFrameFormat_ARROW, pluginv2.DataFrameFormat_JSON} {
		t.Run(format.String(), func(t *testing.T) {
			client := chunkedClient(t, QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, w ChunkedDataWriter) error {
				frame := data.NewFrame("f", data.NewField("v", nil, []int64{1, 2, 3}))
				require.NoError(t, w.WriteFrame(ctx, "A", "f1", frame))
				require.NoError(t, w.WriteFrame(ctx, "A", "f1", frame))
				return w.WriteFrame(ctx, "B", "f2", frame)
			}))

			stream, err := client.QueryChunkedData(context.Background(), &pluginv2.QueryChunkedDataRequest{
				PluginContext: &pluginv2.PluginContext{}, Format: format,
			})
			require.NoError(t, err)

			sum, trailer, streamErr := drain(t, stream)
			require.NoError(t, streamErr)
			require.Positive(t, sum)
			require.Equal(t, strconv.FormatInt(sum, 10), trailer)
		})
	}
}

func TestChunkedDataSetsResponseSizeTrailerWhenTheHandlerFails(t *testing.T) {
	client := chunkedClient(t, QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, w ChunkedDataWriter) error {
		require.NoError(t, w.WriteFrame(ctx, "A", "f1", data.NewFrame("f", data.NewField("v", nil, []int64{1, 2, 3}))))
		return errors.New("boom")
	}))

	stream, err := client.QueryChunkedData(context.Background(), &pluginv2.QueryChunkedDataRequest{PluginContext: &pluginv2.PluginContext{}})
	require.NoError(t, err)

	sum, trailer, streamErr := drain(t, stream)
	require.Error(t, streamErr)
	require.Positive(t, sum)
	require.Equal(t, strconv.FormatInt(sum, 10), trailer, "what was sent before the failure is still counted")
}
