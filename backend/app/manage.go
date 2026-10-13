package app

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-plugin"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
	"github.com/grafana/grafana-plugin-sdk-go/internal/buildinfo"
)

// ManageOpts can modify Manage behavior.
type ManageOpts struct {
	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings

	// TracingOpts contains settings for tracing setup.
	TracingOpts tracing.Opts

	// Stateless admission handler
	AdmissionHandler backend.AdmissionHandler

	// Stateless conversion handler
	ConversionHandler backend.ConversionHandler

	// Attach custom plugins plugins to the gRPC server.
	ExtraPlugins plugin.PluginSet

	// Licensing, if set, is applied to the serve options before the plugin is served.
	// Use marketplace.Licensing from the experimental/marketplace package for marketplace plugins.
	// This is EXPERIMENTAL and is subject to change.
	Licensing backend.ServeOptsDecorator
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
	handler := automanagement.NewManager(NewInstanceManager(instanceFactory))
	serveOpts := backend.ServeOpts{
		CheckHealthHandler:      handler,
		CallResourceHandler:     handler,
		QueryDataHandler:        handler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           handler,
		AdmissionHandler:        opts.AdmissionHandler,
		ConversionHandler:       opts.ConversionHandler,
		GRPCSettings:            opts.GRPCSettings,
		ExtraPlugins:            opts.ExtraPlugins,
	}
	if opts.Licensing != nil {
		var err error
		if serveOpts, err = opts.Licensing(pluginID, serveOpts); err != nil {
			return fmt.Errorf("licensing: %w", err)
		}
	}
	if err := backend.SetupTracer(pluginID, opts.TracingOpts); err != nil {
		return fmt.Errorf("setup tracer: %w", err)
	}
	return backend.Manage(pluginID, serveOpts)
}
