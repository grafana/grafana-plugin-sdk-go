package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
	"github.com/grafana/grafana-plugin-sdk-go/internal/standalone"
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

	serveOpts := backend.ServeOpts{
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		QueryDataHandler:    handler,
		StreamHandler:       handler,
		GRPCSettings:        opts.GRPCSettings,
	}

	info, err := standalone.GetInfo(pluginID)
	if err != nil {
		return err
	}

	if info.Standalone {
		return backend.StandaloneServe(serveOpts, info.Address)
	} else if info.Address != "" {
		standalone.RunDummyPluginLocator(info.Address)
		return nil
	}

	// The default/normal hashicorp path.
	return backend.Serve(serveOpts)
}
