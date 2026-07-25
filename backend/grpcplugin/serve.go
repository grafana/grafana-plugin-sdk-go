package grpcplugin

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// ServeOpts contains options for serving plugins.
type ServeOpts struct {
	DiagnosticsServer DiagnosticsServer
	ResourceServer    ResourceServer
	DataServer        DataServer
	StreamServer      StreamServer
	AdmissionServer   AdmissionServer
	ConversionServer  ConversionServer

	// V3Server serves the grafana.plugin.v3 API — the modern successor to the
	// legacy pluginv2 (backend.proto) contract. Implementations embed
	// UnimplementedV3Server and override the RPCs they support.
	//
	// V3 is additive: it can be set on its own or alongside the legacy *Server
	// fields above, so a single plugin binary can serve both contracts at once
	// and hosts can migrate to v3 incrementally. Each V3 service is dispensed
	// under its own go-plugin name (see pluginSet).
	V3Server V3Server

	// GRPCServer factory method for creating GRPC server.
	// If nil, the default one will be used.
	GRPCServer func(options []grpc.ServerOption) *grpc.Server
}

// pluginSet builds the go-plugin PluginSet from the given options. It is shared
// by Serve and by tests that exercise the go-plugin dispensing path in-process.
func pluginSet(opts ServeOpts) plugin.PluginSet {
	pSet := make(plugin.PluginSet)

	if opts.DiagnosticsServer != nil {
		pSet["diagnostics"] = &DiagnosticsGRPCPlugin{
			DiagnosticsServer: opts.DiagnosticsServer,
		}
	}

	if opts.ResourceServer != nil {
		pSet["resource"] = &ResourceGRPCPlugin{
			ResourceServer: opts.ResourceServer,
		}
	}

	if opts.DataServer != nil {
		pSet["data"] = &DataGRPCPlugin{
			DataServer: opts.DataServer,
		}
	}

	if opts.StreamServer != nil {
		pSet["stream"] = &StreamGRPCPlugin{
			StreamServer: opts.StreamServer,
		}
	}

	if opts.AdmissionServer != nil {
		pSet["admission"] = &AdmissionGRPCPlugin{
			AdmissionServer: opts.AdmissionServer,
		}
	}

	if opts.ConversionServer != nil {
		pSet["conversion"] = &ConversionGRPCPlugin{
			ConversionServer: opts.ConversionServer,
		}
	}

	if opts.V3Server != nil {
		pSet["v3-route"] = &routeGRPCPlugin{server: opts.V3Server}
		pSet["v3-admission-control"] = &admissionControlGRPCPlugin{server: opts.V3Server}
		pSet["v3-resource-conversion"] = &resourceConversionGRPCPlugin{server: opts.V3Server}
	}

	return pSet
}

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
	versionedPlugins := make(map[int]plugin.PluginSet)
	pSet := pluginSet(opts)
	versionedPlugins[ProtocolVersion] = pSet

	if opts.GRPCServer == nil {
		opts.GRPCServer = plugin.DefaultGRPCServer
	}

	plugKeys := make([]string, 0, len(pSet))
	for k := range pSet {
		plugKeys = append(plugKeys, k)
	}
	log.DefaultLogger.Debug("Serving plugin", "plugins", plugKeys)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  handshake,
		VersionedPlugins: versionedPlugins,
		GRPCServer:       opts.GRPCServer,
	})
	log.DefaultLogger.Debug("Plugin server exited")

	return nil
}
