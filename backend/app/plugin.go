package app

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// ServeOpts options for serving an app plugin.
type ServeOpts struct {
	// CheckHealthHandler handler for health checks.
	// Optional to implement.
	backend.CheckHealthHandler

	// CallResourceHandler handler for resource calls.
	// Optional to implement.
	backend.CallResourceHandler

	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings
}

// Serve starts serving the app over gPRC.
func Serve(opts ServeOpts) error {
	return backend.Serve(backend.ServeOpts{
		CheckHealthHandler:  opts.CheckHealthHandler,
		CallResourceHandler: opts.CallResourceHandler,
		GRPCSettings:        opts.GRPCSettings,
	})
}
