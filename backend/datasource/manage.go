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

	// When an MCP server is present, wrap the gRPC handler so that every
	// incoming Grafana call registers the datasource PluginContext. MCP tool
	// calls then look up the context by UID instead of receiving an empty one.
	grpcHandler := PluginHandler(handler)
	if opts.MCPServer != nil {
		grpcHandler = &contextCapture{inner: handler, mcpServer: opts.MCPServer}
	}

	return backend.Manage(pluginID, backend.ServeOpts{
		CheckHealthHandler:      grpcHandler,
		CallResourceHandler:     grpcHandler,
		QueryDataHandler:        grpcHandler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           grpcHandler,
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

// contextCapture wraps a PluginHandler and registers each datasource's
// PluginContext with the MCP server as gRPC calls arrive from Grafana.
// This lets MCP tool calls look up the context (including decrypted credentials)
// by datasource UID without manual configuration.
type contextCapture struct {
	inner     PluginHandler
	mcpServer *mcp.Server
}

func (c *contextCapture) capture(pctx backend.PluginContext) {
	if pctx.DataSourceInstanceSettings != nil {
		c.mcpServer.RegisterPluginContext(pctx.DataSourceInstanceSettings.UID, pctx)
	}
}

func (c *contextCapture) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	c.capture(req.PluginContext)
	return c.inner.QueryData(ctx, req)
}

func (c *contextCapture) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	c.capture(req.PluginContext)
	return c.inner.CallResource(ctx, req, sender)
}

func (c *contextCapture) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	c.capture(req.PluginContext)
	return c.inner.CheckHealth(ctx, req)
}

func (c *contextCapture) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	return c.inner.SubscribeStream(ctx, req)
}

func (c *contextCapture) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return c.inner.PublishStream(ctx, req)
}

func (c *contextCapture) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	return c.inner.RunStream(ctx, req, sender)
}
