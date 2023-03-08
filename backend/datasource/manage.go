package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
)

// ManageOpts can modify Manage behaviour.
type ManageOpts struct {
	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings
}

// Manage starts serving the data source over gPRC with automatic instance management.
// pluginID should match the one from plugin.json.
func Manage(pluginID string, instanceFactory InstanceFactoryFunc, opts ManageOpts) error {
	backend.SetupPluginEnvironment(pluginID) // Enable profiler.
	handler := automanagement.NewManager(NewInstanceManager(instanceFactory))
	return backend.Manage(pluginID, backend.ServeOpts{
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		QueryDataHandler:    handler,
		StreamHandler:       handler,
		GRPCSettings:        opts.GRPCSettings,
	})
}
