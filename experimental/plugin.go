package experimental

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// PluginSingleton is a singleton instance
type PluginSingleton interface {
	// CheckHostHealth for the plugin executable
	CheckHostHealth(config backend.PluginConfig) *backend.CheckHealthResult

	// request for a new datasource
	NewDataSourceInstance(config backend.PluginConfig) (DataSourceInstance, error)
}

// DataSourceInstance will get created for each org/id and then regenerated when the lastModified times change
type DataSourceInstance interface {
	CheckHealth() *backend.CheckHealthResult

	// If the request does not need access to the headers or user, use this request
	QueryData(req *backend.QueryDataRequest) (*backend.QueryDataResponse, error)

	// Get resource
	CallResource(req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error

	// Destroy lets you clean up any instance variables when the settings change
	Destroy()
}
