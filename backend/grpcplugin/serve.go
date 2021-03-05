package grpcplugin

import (
	"net"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// ServeOpts contains options for serving plugins.
type ServeOpts struct {
	DiagnosticsServer DiagnosticsServer
	ResourceServer    ResourceServer
	DataServer        DataServer
	TransformServer   TransformServer

	// GRPCServer factory method for creating GRPC server.
	// If nil, the default one will be used.
	GRPCServer func(options []grpc.ServerOption) *grpc.Server
}

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
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

	if opts.TransformServer != nil {
		// pluginv2.RegisterTransformServer(server, opts.TransformServer)
	}

	log.DefaultLogger.Debug("Serving plugin", "plugins", plugKeys)

	listener, err := net.Listen("tcp", ":3021")
	if err != nil {
		return err
	}

	server.Serve(listener)
	log.DefaultLogger.Debug("Plugin server exited")

	return nil
}
