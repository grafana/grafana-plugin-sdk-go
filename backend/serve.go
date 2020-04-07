package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	maxServerReceiveMsgSize = 1024 * 1024 * 4
	maxServerSendMsgSize    = 1024 * 1024 * 4
)

//ServeOpts options for serving plugins.
type ServeOpts struct {
	CheckHealthHandler   CheckHealthHandler
	CallResourceHandler  CallResourceHandler
	QueryDataHandler     QueryDataHandler
	TransformDataHandler TransformDataHandler
}

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
	pluginOpts := grpcplugin.ServeOpts{
		DiagnosticsServer: newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, opts.CheckHealthHandler),
	}

	if opts.CallResourceHandler != nil {
		pluginOpts.ResourceServer = newResourceSDKAdapter(opts.CallResourceHandler)
	}

	if opts.QueryDataHandler != nil {
		pluginOpts.DataServer = newDataSDKAdapter(opts.QueryDataHandler)
	}

	if opts.TransformDataHandler != nil {
		pluginOpts.TransformServer = newTransformSDKAdapter(opts.TransformDataHandler)
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	recoveryOption := grpc_recovery.WithRecoveryHandlerContext(recoveryHandler)

	pluginOpts.GRPCServer = func(opts []grpc.ServerOption) *grpc.Server {
		mergedOpts := []grpc.ServerOption{}
		mergedOpts = append(mergedOpts, opts...)
		sopts := []grpc.ServerOption{
			grpc.MaxRecvMsgSize(maxServerReceiveMsgSize),
			grpc.MaxSendMsgSize(maxServerSendMsgSize),
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
				grpc_prometheus.StreamServerInterceptor,
				grpc_recovery.StreamServerInterceptor(recoveryOption),
			)),
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
				grpc_prometheus.UnaryServerInterceptor,
				grpc_recovery.UnaryServerInterceptor(recoveryOption),
			)),
		}
		mergedOpts = append(mergedOpts, sopts...)
		return grpc.NewServer(mergedOpts...)
	}

	return grpcplugin.Serve(pluginOpts)
}
