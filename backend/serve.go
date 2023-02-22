package backend

import (
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

const defaultServerMaxReceiveMessageSize = 1024 * 1024 * 16

// GRPCSettings settings for gRPC.
type GRPCSettings struct {
	// MaxReceiveMsgSize the max gRPC message size in bytes the plugin can receive.
	// If this is <= 0, gRPC uses the default 16MB.
	MaxReceiveMsgSize int

	// MaxSendMsgSize the max gRPC message size in bytes the plugin can send.
	// If this is <= 0, gRPC uses the default `math.MaxInt32`.
	MaxSendMsgSize int
}

// ServeOpts options for serving plugins.
type ServeOpts struct {
	// CheckHealthHandler handler for health checks.
	CheckHealthHandler CheckHealthHandler

	// CallResourceHandler handler for resource calls.
	// Optional to implement.
	CallResourceHandler CallResourceHandler

	// QueryDataHandler handler for data queries.
	// Required to implement if data source.
	QueryDataHandler QueryDataHandler

	// StreamHandler handler for streaming queries.
	// This is EXPERIMENTAL and is a subject to change till Grafana 8.
	StreamHandler StreamHandler

	// GRPCSettings settings for gPRC.
	GRPCSettings GRPCSettings
}

func asGRPCServeOpts(opts ServeOpts) grpcplugin.ServeOpts {
	pluginOpts := grpcplugin.ServeOpts{
		DiagnosticsServer: newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, opts.CheckHealthHandler),
	}

	if opts.CallResourceHandler != nil {
		pluginOpts.ResourceServer = newResourceSDKAdapter(opts.CallResourceHandler)
	}

	if opts.QueryDataHandler != nil {
		pluginOpts.DataServer = newDataSDKAdapter(opts.QueryDataHandler)
	}

	if opts.StreamHandler != nil {
		pluginOpts.StreamServer = newStreamSDKAdapter(opts.StreamHandler)
	}
	return pluginOpts
}

func defaultGRPCMiddlewares(opts ServeOpts) []grpc.ServerOption {
	if opts.GRPCSettings.MaxReceiveMsgSize <= 0 {
		opts.GRPCSettings.MaxReceiveMsgSize = defaultServerMaxReceiveMessageSize
	}
	grpcMiddlewares := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(opts.GRPCSettings.MaxReceiveMsgSize),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			otelgrpc.StreamServerInterceptor(),
			grpc_opentracing.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			otelgrpc.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}
	if opts.GRPCSettings.MaxSendMsgSize > 0 {
		grpcMiddlewares = append([]grpc.ServerOption{grpc.MaxSendMsgSize(opts.GRPCSettings.MaxSendMsgSize)}, grpcMiddlewares...)
	}
	return grpcMiddlewares
}

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
	grpc_prometheus.EnableHandlingTimeHistogram()
	pluginOpts := asGRPCServeOpts(opts)
	pluginOpts.GRPCServer = func(grpcOptions []grpc.ServerOption) *grpc.Server {
		return grpc.NewServer(append(defaultGRPCMiddlewares(opts), grpcOptions...)...)
	}
	return grpcplugin.Serve(pluginOpts)
}

// StandaloneServe starts a gRPC server that is not managed by hashicorp
func StandaloneServe(opts ServeOpts, address string) error {
	pluginOpts := asGRPCServeOpts(opts)
	if pluginOpts.GRPCServer == nil {
		pluginOpts.GRPCServer = func(grpcOptions []grpc.ServerOption) *grpc.Server {
			return grpc.NewServer(append(defaultGRPCMiddlewares(opts), grpcOptions...)...)
		}
	}

	server := pluginOpts.GRPCServer(nil)
	plugKeys := []string{}
	if pluginOpts.DiagnosticsServer != nil {
		pluginv2.RegisterDiagnosticsServer(server, pluginOpts.DiagnosticsServer)
		plugKeys = append(plugKeys, "diagnostics")
	}

	if pluginOpts.ResourceServer != nil {
		pluginv2.RegisterResourceServer(server, pluginOpts.ResourceServer)
		plugKeys = append(plugKeys, "resources")
	}

	if pluginOpts.DataServer != nil {
		pluginv2.RegisterDataServer(server, pluginOpts.DataServer)
		plugKeys = append(plugKeys, "data")
	}

	if pluginOpts.StreamServer != nil {
		pluginv2.RegisterStreamServer(server, pluginOpts.StreamServer)
		plugKeys = append(plugKeys, "stream")
	}

	log.DefaultLogger.Debug("Standalone plugin server", "capabilities", plugKeys)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	err = server.Serve(listener)
	if err != nil {
		return err
	}
	log.DefaultLogger.Debug("Plugin server exited")

	return nil
}
