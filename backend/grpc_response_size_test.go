package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestSplitGRPCFullMethod(t *testing.T) {
	tcs := []struct {
		in             string
		wantService    string
		wantMethod     string
	}{
		{"/pluginv2.Data/QueryData", "pluginv2.Data", "QueryData"},
		{"/pluginv2.Resource/CallResource", "pluginv2.Resource", "CallResource"},
		{"/pluginv2.Diagnostics/CheckHealth", "pluginv2.Diagnostics", "CheckHealth"},
		{"", "", ""},
		{"bogus", "", "bogus"},
	}
	for _, tc := range tcs {
		t.Run(tc.in, func(t *testing.T) {
			svc, m := splitGRPCFullMethod(tc.in)
			require.Equal(t, tc.wantService, svc)
			require.Equal(t, tc.wantMethod, m)
		})
	}
}

func TestGRPCResponseSizeInterceptor(t *testing.T) {
	interceptor := grpcResponseSizeInterceptor()

	t.Run("observes size of successful proto response", func(t *testing.T) {
		const service, method = "pluginv2.Diagnostics", "CheckHealth"
		resp := &pluginv2.CheckHealthResponse{
			Status:  pluginv2.CheckHealthResponse_OK,
			Message: "healthy-ish",
		}
		expectedSize := float64(proto.Size(resp))

		before := sampleCount(t, service, method)
		_, err := interceptor(context.Background(), nil,
			&grpc.UnaryServerInfo{FullMethod: "/" + service + "/" + method},
			func(_ context.Context, _ any) (any, error) { return resp, nil },
		)
		require.NoError(t, err)

		got := latestSample(t, service, method)
		require.Equal(t, before+1, got.count, "exactly one new observation")
		require.Equal(t, expectedSize, got.sumDelta, "observed value matches proto.Size")
	})

	t.Run("skips observation when handler returns error", func(t *testing.T) {
		const service, method = "pluginv2.Data", "QueryData"
		before := sampleCount(t, service, method)
		_, err := interceptor(context.Background(), nil,
			&grpc.UnaryServerInfo{FullMethod: "/" + service + "/" + method},
			func(_ context.Context, _ any) (any, error) {
				return &pluginv2.QueryDataResponse{}, errors.New("boom")
			},
		)
		require.Error(t, err)
		require.Equal(t, before, sampleCount(t, service, method))
	})

	t.Run("skips observation on nil response", func(t *testing.T) {
		const service, method = "pluginv2.Resource", "CallResource"
		before := sampleCount(t, service, method)
		_, err := interceptor(context.Background(), nil,
			&grpc.UnaryServerInfo{FullMethod: "/" + service + "/" + method},
			func(_ context.Context, _ any) (any, error) { return nil, nil },
		)
		require.NoError(t, err)
		require.Equal(t, before, sampleCount(t, service, method))
	})

	t.Run("skips observation on non-proto response", func(t *testing.T) {
		const service, method = "test.Svc", "NonProto"
		before := sampleCount(t, service, method)
		_, err := interceptor(context.Background(), nil,
			&grpc.UnaryServerInfo{FullMethod: "/" + service + "/" + method},
			func(_ context.Context, _ any) (any, error) { return "not a proto.Message", nil },
		)
		require.NoError(t, err)
		require.Equal(t, before, sampleCount(t, service, method))
	})
}

type histSample struct {
	count    uint64
	sumDelta float64
}

func histogramDTO(t *testing.T, service, method string) *dto.Histogram {
	t.Helper()
	ch := make(chan prometheus.Metric, 1)
	grpcResponseSizeHistogram.WithLabelValues(service, method).(prometheus.Histogram).Collect(ch)
	close(ch)
	m := <-ch
	var pb dto.Metric
	require.NoError(t, m.Write(&pb))
	return pb.GetHistogram()
}

func sampleCount(t *testing.T, service, method string) uint64 {
	t.Helper()
	return histogramDTO(t, service, method).GetSampleCount()
}

func latestSample(t *testing.T, service, method string) histSample {
	t.Helper()
	h := histogramDTO(t, service, method)
	return histSample{count: h.GetSampleCount(), sumDelta: h.GetSampleSum()}
}
