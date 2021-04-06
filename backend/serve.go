package backend

import (
	"net"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/hashicorp/go-plugin"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
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

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
	grpc_prometheus.EnableHandlingTimeHistogram()
	grpcMiddlewares := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}

	if opts.GRPCSettings.MaxReceiveMsgSize <= 0 {
		opts.GRPCSettings.MaxReceiveMsgSize = defaultServerMaxReceiveMessageSize
	}

	grpcMiddlewares = append([]grpc.ServerOption{grpc.MaxRecvMsgSize(opts.GRPCSettings.MaxReceiveMsgSize)}, grpcMiddlewares...)

	if opts.GRPCSettings.MaxSendMsgSize > 0 {
		grpcMiddlewares = append([]grpc.ServerOption{grpc.MaxSendMsgSize(opts.GRPCSettings.MaxSendMsgSize)}, grpcMiddlewares...)
	}

	pluginOpts := asGRPCServeOpts(opts)
	pluginOpts.GRPCServer = func(opts []grpc.ServerOption) *grpc.Server {
		opts = append(opts, grpcMiddlewares...)
		return grpc.NewServer(opts...)
	}

	return grpcplugin.Serve(pluginOpts)
}

// StandaloneServe starts a gRPC server that is not managed by hashicorp
func StandaloneServe(dsopts ServeOpts, address string) error {
	opts := asGRPCServeOpts(dsopts)

	if opts.GRPCServer == nil {
		opts.GRPCServer = plugin.DefaultGRPCServer
	}

	server := opts.GRPCServer(nil)

	plugKeys := []string{}
	if opts.DiagnosticsServer != nil {
		pluginv2.RegisterDiagnosticsServer(server, opts.DiagnosticsServer)
		plugKeys = append(plugKeys, "diagnostics")
	}

	if opts.ResourceServer != nil {
		pluginv2.RegisterResourceServer(server, opts.ResourceServer)
		plugKeys = append(plugKeys, "resources")
	}

	if opts.DataServer != nil {
		pluginv2.RegisterDataServer(server, opts.DataServer)
		plugKeys = append(plugKeys, "data")
	}

	if opts.StreamServer != nil {
		pluginv2.RegisterStreamServer(server, opts.StreamServer)
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
