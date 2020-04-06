package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
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

	pluginOpts.GRPCServer = func(opts []grpc.ServerOption) *grpc.Server {
		opts = append(opts, grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor))
		opts = append(opts, grpc.UnaryInterceptor(grpcprometheus.UnaryServerInterceptor))
		return grpc.NewServer(opts...)
	}

	return grpcplugin.Serve(pluginOpts)
}
