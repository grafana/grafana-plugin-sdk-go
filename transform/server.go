package transform

import (
	"github.com/grafana/grafana-plugin-sdk-go/common"
	plugin "github.com/hashicorp/go-plugin"
)

// PluginName the name of the plugin that can be dispensed
// from the plugin server.
const PluginName = "transform"

// Serve starts serving the transform plugin over gRPC.
func Serve(handler TransformHandler) error {
	versionedPlugins := map[int]plugin.PluginSet{
		common.ProtocolVersion: {
			PluginName: &TransformPluginImpl{
				Impl: transformPluginWrapper{
					handler: handler,
				},
			},
		},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  common.Handshake,
		VersionedPlugins: versionedPlugins,
		GRPCServer:       plugin.DefaultGRPCServer,
	})

	return nil
}
