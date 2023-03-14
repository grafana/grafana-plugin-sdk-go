package datasource

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
)

// ManageOpts can modify Manage behaviour.
type ManageOpts struct {
	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings

	// TracingOpts contains settings for tracing setup.
	TracingOpts tracing.Opts
}

// Manage starts serving the data source over gPRC with automatic instance management.
// pluginID should match the one from plugin.json.
func Manage(pluginID string, instanceFactory InstanceFactoryFunc, opts ManageOpts) error {
	// Enable profiler.
	backend.SetupPluginEnvironment(pluginID)

	// Set up tracing
	// TODO: replicate in app as well
	tracingCfg := backend.GetTracingConfig()
	if tracingCfg.IsEnabled() {
		tp, err := tracing.NewTraceProvider(tracingCfg.Address, pluginID, opts.TracingOpts)
		if err != nil {
			return fmt.Errorf("new trace provider: %w", err)
		}
		tracing.InitGlobalTraceProvider(tp, tracing.NewPropagatorFormat(tracingCfg.Propagation))
	}
	backend.Logger.Info("Tracing", "enabled", tracingCfg.IsEnabled(), "propagation", tracingCfg.Propagation)

	handler := automanagement.NewManager(NewInstanceManager(instanceFactory))
	return backend.Manage(pluginID, backend.ServeOpts{
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		QueryDataHandler:    handler,
		StreamHandler:       handler,
		GRPCSettings:        opts.GRPCSettings,
	})
}
