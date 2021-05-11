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
func Manage(factory InstanceFactoryFunc, opts ManageOpts) error {
	handler := automanagement.NewManager(NewInstanceManager(factory))
	return backend.Serve(backend.ServeOpts{
		QueryDataHandler:    handler,
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		StreamHandler:       handler,
		GRPCSettings:        opts.GRPCSettings,
	})
}
