package backend

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	grpcRespSizeKB = 1024
	grpcRespSizeMB = grpcRespSizeKB * grpcRespSizeKB
	grpcRespSizeGB = grpcRespSizeMB * grpcRespSizeKB
)

var grpcResponseSizeHistogram = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "plugins",
		Name:      "grpc_response_size_bytes",
		Help:      "Histogram of plugin gRPC response message sizes (uncompressed protobuf bytes) sent from the plugin to Grafana.",
		Buckets: []float64{
			128, 256, 512,
			1 * grpcRespSizeKB, 2 * grpcRespSizeKB, 4 * grpcRespSizeKB, 8 * grpcRespSizeKB,
			16 * grpcRespSizeKB, 32 * grpcRespSizeKB, 64 * grpcRespSizeKB, 128 * grpcRespSizeKB,
			256 * grpcRespSizeKB, 512 * grpcRespSizeKB,
			1 * grpcRespSizeMB, 2 * grpcRespSizeMB, 4 * grpcRespSizeMB, 8 * grpcRespSizeMB,
			16 * grpcRespSizeMB, 32 * grpcRespSizeMB, 64 * grpcRespSizeMB, 128 * grpcRespSizeMB,
			256 * grpcRespSizeMB, 512 * grpcRespSizeMB,
			1 * grpcRespSizeGB, 2 * grpcRespSizeGB, 4 * grpcRespSizeGB, 8 * grpcRespSizeGB,
		},
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	},
	[]string{"grpc_service", "grpc_method"},
)

// grpcResponseSizeInterceptor returns a unary server interceptor that
// observes the uncompressed protobuf size of every gRPC response sent
// from the plugin. Responses are only observed on successful handler
// returns; error responses are skipped.
func grpcResponseSizeInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil || resp == nil {
			return resp, err
		}
		msg, ok := resp.(proto.Message)
		if !ok {
			return resp, err
		}
		service, method := splitGRPCFullMethod(info.FullMethod)
		grpcResponseSizeHistogram.
			WithLabelValues(service, method).
			Observe(float64(proto.Size(msg)))
		return resp, err
	}
}

// splitGRPCFullMethod parses grpc.UnaryServerInfo.FullMethod ("/service/method")
// into its service and method components. Falls back to empty strings on
// unexpected formats rather than failing observation.
func splitGRPCFullMethod(full string) (service, method string) {
	s := strings.TrimPrefix(full, "/")
	if svc, m, ok := strings.Cut(s, "/"); ok {
		return svc, m
	}
	return "", s
}
