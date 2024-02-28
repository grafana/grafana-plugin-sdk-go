package app

import (
	"fmt"
	"log"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
	"github.com/grafana/grafana-plugin-sdk-go/internal/buildinfo"
)

// ManageOpts can modify Manage behaviour.
type ManageOpts struct {
	// ApiVersion is the required ApiVersion -- setting this will
	// fail any request asking for a different version.  For multi-version servers,
	// this value should not be set, and be handled directly in each server
	ApiVersion string

	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings

	// TracingOpts contains settings for tracing setup.
	TracingOpts tracing.Opts
}

// Manage starts serving the app over gPRC with automatic instance management.
// pluginID should match the one from plugin.json.
func Manage(pluginID string, instanceFactory InstanceFactoryFunc, opts ManageOpts) error {
	// If we are running in build info mode, run that and exit
	if buildinfo.InfoModeEnabled() {
		if err := buildinfo.RunInfoMode(); err != nil {
			log.Fatalln(err)
			return err
		}
		os.Exit(0)
		return nil
	}

	backend.SetupPluginEnvironment(pluginID)
	if err := backend.SetupTracer(pluginID, opts.TracingOpts); err != nil {
		return fmt.Errorf("setup tracer: %w", err)
	}
	handler := automanagement.NewManager(NewInstanceManager(instanceFactory))
	return backend.Manage(pluginID, backend.ServeOpts{
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		QueryDataHandler:    handler,
		StreamHandler:       handler,
		GRPCSettings:        opts.GRPCSettings,
	})
}
