package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/common"
	plugin "github.com/hashicorp/go-plugin"
)

// Serve starts serving the datasource plugin over gRPC.
//
// The plugin ID should be in the format <org>-<name>-datasource.
func Serve(pluginID string, backendHandlers *PluginHandlers, platformHandlers *PlatformHandlers) error {
	versionedPlugins := make(map[int]plugin.PluginSet)

	pSet := make(plugin.PluginSet)
	if backendHandlers != nil {
		pSet["backend"] = &CoreImpl{
			Wrap: coreWrapper{
				handlers: *backendHandlers,
			},
		}
	}

	if backendHandlers != nil {
		pSet["platform"] = &PlatformImpl{
			Wrap: platformWrapper{
				Handlers: *platformHandlers,
			},
		}
	}

	versionedPlugins[common.ProtocolVersion] = pSet

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  common.Handshake,
		VersionedPlugins: versionedPlugins,
		GRPCServer:       plugin.DefaultGRPCServer,
	})

	return nil
}
