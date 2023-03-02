package backend

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/hashicorp/go-plugin"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/grafana/grafana-plugin-sdk-go/internal/standalone"
)

const defaultServerMaxReceiveMessageSize = 1024 * 1024 * 16

// GRPCSettings settings for gRPC.
type GRPCSettings struct {
	// MaxReceiveMsgSize the max gRPC message size in bytes the plugin can receive.
	// If this is <= 0, gRPC uses the default 16MB.
	MaxReceiveMsgSize int

	// MaxSendMsgSize the max gRPC message size in bytes the plugin can send.
	// If this is <= 0, gRPC uses the default `math.MaxInt32`.
	MaxSendMsgSize int
}

// ServeOpts options for serving plugins.
type ServeOpts struct {
	// CheckHealthHandler handler for health checks.
	CheckHealthHandler CheckHealthHandler

	// CallResourceHandler handler for resource calls.
	// Optional to implement.
	CallResourceHandler CallResourceHandler

	// QueryDataHandler handler for data queries.
	// Required to implement if data source.
	QueryDataHandler QueryDataHandler

	// StreamHandler handler for streaming queries.
	// This is EXPERIMENTAL and is a subject to change till Grafana 8.
	StreamHandler StreamHandler

	// GRPCSettings settings for gPRC.
	GRPCSettings GRPCSettings
}

func asGRPCServeOpts(opts ServeOpts) grpcplugin.ServeOpts {
	pluginOpts := grpcplugin.ServeOpts{
		DiagnosticsServer: newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, opts.CheckHealthHandler),
	}

	if opts.CallResourceHandler != nil {
		pluginOpts.ResourceServer = newResourceSDKAdapter(opts.CallResourceHandler)
	}

	if opts.QueryDataHandler != nil {
		pluginOpts.DataServer = newDataSDKAdapter(opts.QueryDataHandler)
	}

	if opts.StreamHandler != nil {
		pluginOpts.StreamServer = newStreamSDKAdapter(opts.StreamHandler)
	}
	return pluginOpts
}

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
	grpc_prometheus.EnableHandlingTimeHistogram()
	grpcMiddlewares := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}

	if opts.GRPCSettings.MaxReceiveMsgSize <= 0 {
		opts.GRPCSettings.MaxReceiveMsgSize = defaultServerMaxReceiveMessageSize
	}

	grpcMiddlewares = append([]grpc.ServerOption{grpc.MaxRecvMsgSize(opts.GRPCSettings.MaxReceiveMsgSize)}, grpcMiddlewares...)

	if opts.GRPCSettings.MaxSendMsgSize > 0 {
		grpcMiddlewares = append([]grpc.ServerOption{grpc.MaxSendMsgSize(opts.GRPCSettings.MaxSendMsgSize)}, grpcMiddlewares...)
	}

	pluginOpts := asGRPCServeOpts(opts)
	pluginOpts.GRPCServer = func(opts []grpc.ServerOption) *grpc.Server {
		opts = append(opts, grpcMiddlewares...)
		return grpc.NewServer(opts...)
	}

	return grpcplugin.Serve(pluginOpts)
}

// StandaloneServe starts a gRPC server that is not managed by hashicorp.
// Deprecated: use GracefulStandaloneServe instead.
func StandaloneServe(dsopts ServeOpts, address string) error {
	// GracefulStandaloneServe has a new signature, this function keeps the old
	// signature for existing plugins for backwards compatibility.
	// Create a new standalone.Args and disable all the standalone-file-related features.
	return GracefulStandaloneServe(dsopts, standalone.Args{Address: address})
}

// GracefulStandaloneServe starts a gRPC server that is not managed by hashicorp.
// The provided standalone.Args must have an Address set, or the function returns an error.
// The function handles creating/cleaning up the standalone address file, and graceful GRPC server termination.
// The function returns after the GRPC server has been terminated.
func GracefulStandaloneServe(dsopts ServeOpts, info standalone.Args) error {
	// We must have an address if we want to run the plugin in standalone mode
	if info.Address == "" {
		return fmt.Errorf("standalone address must be specified")
	}

	// Write the address to the local file
	if info.Debugger {
		log.DefaultLogger.Info("Creating standalone address and pid files")
		if err := standalone.CreateStandaloneAddressFile(info); err != nil {
			return fmt.Errorf("create standalone address file: %w", err)
		}
		if err := standalone.CreateStandalonePIDFile(info); err != nil {
			return fmt.Errorf("create standalone pid file: %w", err)
		}

		// sadly vs-code can not listen to shutdown events
		// https://github.com/golang/vscode-go/issues/120

		// Cleanup function that deletes standalone.txt and pid.txt, if it exists. Fails silently.
		// This is so the address file is deleted when the plugin shuts down gracefully, if possible.
		defer func() {
			log.DefaultLogger.Info("Cleaning up standalone address and pid files")
			if err := standalone.CleanupStandaloneAddressFile(info); err != nil {
				log.DefaultLogger.Error("Error while cleaning up standalone address file", "error", err)
			}
			if err := standalone.CleanupStandalonePIDFile(info); err != nil {
				log.DefaultLogger.Error("Error while cleaning up standalone pid file", "error", err)
			}
			// Kill the dummy locator so Grafana reloads the plugin
			standalone.FindAndKillCurrentPlugin(info.Dir)
		}()

		// When debugging, be sure to kill the running instances, so we reconnect
		standalone.FindAndKillCurrentPlugin(info.Dir)
	}

	// Start GRPC server
	opts := asGRPCServeOpts(dsopts)
	if opts.GRPCServer == nil {
		opts.GRPCServer = plugin.DefaultGRPCServer
	}

	server := opts.GRPCServer(nil)

	var plugKeys []string
	if opts.DiagnosticsServer != nil {
		pluginv2.RegisterDiagnosticsServer(server, opts.DiagnosticsServer)
		plugKeys = append(plugKeys, "diagnostics")
	}

	if opts.ResourceServer != nil {
		pluginv2.RegisterResourceServer(server, opts.ResourceServer)
		plugKeys = append(plugKeys, "resources")
	}

	if opts.DataServer != nil {
		pluginv2.RegisterDataServer(server, opts.DataServer)
		plugKeys = append(plugKeys, "data")
	}

	if opts.StreamServer != nil {
		pluginv2.RegisterStreamServer(server, opts.StreamServer)
		plugKeys = append(plugKeys, "stream")
	}

	// Start the GRPC server and handle graceful shutdown to ensure we execute deferred functions correctly
	log.DefaultLogger.Debug("Standalone plugin server", "capabilities", plugKeys)
	listener, err := net.Listen("tcp", info.Address)
	if err != nil {
		return err
	}

	signalChan := make(chan os.Signal, 1)
	serverErrChan := make(chan error, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Unregister signal handlers before returning
	defer signal.Stop(signalChan)

	// Start GRPC server in a separate goroutine
	go func() {
		serverErrChan <- server.Serve(listener)
	}()

	// Block until signal or GRPC server termination
	select {
	case <-signalChan:
		// Signal received, stop the server
		server.Stop()
		if err := <-serverErrChan; err != nil {
			// Bubble up error
			return err
		}
	case err := <-serverErrChan:
		// Server stopped prematurely, bubble up the error
		return err
	}

	log.DefaultLogger.Debug("Plugin server exited")
	return nil
}

// Manage runs the plugin in either standalone mode, dummy locator or normal (hashicorp) mode.
func Manage(pluginID string, serveOpts ServeOpts) error {
	info, err := standalone.GetInfo(pluginID)
	if err != nil {
		return err
	}

	if info.Standalone {
		// Run the standalone GRPC server
		return GracefulStandaloneServe(serveOpts, info)
	}

	if info.Address != "" && (info.PID == 0 || standalone.CheckPIDIsRunning(info.PID)) {
		// Grafana is trying to run the dummy plugin locator to connect to the standalone
		// GRPC server (separate process)
		Logger.Debug("Running dummy plugin locator", "addr", info.Address, "pid", strconv.Itoa(info.PID))
		standalone.RunDummyPluginLocator(info.Address)
		return nil
	}

	// The default/normal hashicorp path.
	return Serve(serveOpts)
}
