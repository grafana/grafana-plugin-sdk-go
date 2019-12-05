package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/common"
	plugin "github.com/hashicorp/go-plugin"
)

// PluginName the name of the data source plugin that can be dispensed
// from the plugin server.
const PluginName = "datasource"

// Serve starts serving the datasource plugin over gRPC.
func Serve(handler DataSourceHandler) error {
	versionedPlugins := map[int]plugin.PluginSet{
		common.ProtocolVersion: {
			PluginName: &DatasourcePluginImpl{
				Impl: datasourcePluginWrapper{
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
