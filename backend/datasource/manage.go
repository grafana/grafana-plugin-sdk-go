package datasource

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	sdklog "github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
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

	// Stateless query conversion handler
	QueryConversionHandler backend.QueryConversionHandler

	// MCPServer, if non-nil, is started alongside the gRPC plugin server and
	// shut down on plugin termination. MCP startup failures are logged but do
	// not prevent the plugin from running.
	MCPServer *mcp.Server
}

// Manage starts serving the data source over gPRC with automatic instance management.
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

	if opts.MCPServer != nil {
		if err := startMCPServer(opts.MCPServer); err != nil {
			sdklog.DefaultLogger.Warn("MCP server startup error", "err", err)
		}
		defer func() {
			if err := stopMCPServer(opts.MCPServer); err != nil {
				sdklog.DefaultLogger.Warn("MCP server shutdown error", "err", err)
			}
		}()
	}

	return backend.Manage(pluginID, backend.ServeOpts{
		CheckHealthHandler:      handler,
		CallResourceHandler:     handler,
		QueryDataHandler:        handler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           handler,
		QueryConversionHandler:  opts.QueryConversionHandler,
		AdmissionHandler:        opts.AdmissionHandler,
		GRPCSettings:            opts.GRPCSettings,
		ConversionHandler:       opts.ConversionHandler,
	})
}

// startMCPServer starts the MCP server. Errors are logged and swallowed; the
// plugin continues without MCP if startup fails (e.g. port in use).
func startMCPServer(s *mcp.Server) error {
	if err := s.Start(context.Background()); err != nil {
		sdklog.DefaultLogger.Warn("MCP server failed to start - continuing without MCP", "err", err)
		return nil
	}
	return nil
}

func stopMCPServer(s *mcp.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}

// PluginHandler is the union of handler interfaces an automanagement-backed
// plugin handler implements. Plugins use this to bind the same handler to
// both gRPC (via Manage) and MCP (via mcp.Server.Bind*).
type PluginHandler interface {
	backend.QueryDataHandler
	backend.CallResourceHandler
	backend.CheckHealthHandler
	backend.StreamHandler
}

// NewAutomanagementHandler wraps the given InstanceManager with the SDK's
// automanagement layer and returns the resulting handler. The returned value
// implements PluginHandler. Plugins bind this to mcp.Server then pass the same
// instance factory through datasource.Manage.
func NewAutomanagementHandler(im instancemgmt.InstanceManager) PluginHandler {
	return automanagement.NewManager(im)
}
