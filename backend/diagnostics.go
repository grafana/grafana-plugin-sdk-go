package backend

import (
	"context"
)

// CheckHealthHandler enables users to send health check
// requests to a backend plugin
type CheckHealthHandler interface {
	CheckPluginHealth(ctx context.Context, req *CheckPluginHealthRequest) (*CheckHealthResult, error)
	CheckDatasourceHealth(ctx context.Context, req *CheckDatasourceHealthRequest) (*CheckHealthResult, error)
}

// HealthStatus is the status of the plugin.
type HealthStatus int

const (
	// HealthStatusUnknown means the status of the plugin is unknown.
	HealthStatusUnknown HealthStatus = iota
	// HealthStatusOk means the status of the plugin is good.
	HealthStatusOk
	// HealthStatusError means the plugin is in an error state.
	HealthStatusError
)

// CheckPluginHealthRequest contains the healthcheck request.
type CheckPluginHealthRequest struct {
	PluginConfig PluginConfig
}

// CheckDatasourceHealth requests contains the healthcheck for an datasource.
type CheckDatasourceHealthRequest struct {
	DataSourceConfig DataSourceConfig
}

// CheckHealthResult contains the healthcheck response
type CheckHealthResult struct {
	Status      HealthStatus
	Message     string
	JSONDetails string
}
