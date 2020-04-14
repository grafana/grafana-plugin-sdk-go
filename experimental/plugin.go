package experimental

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// PluginHost is the singleton container for your plugin.
type PluginHost interface {
	// CheckExeHealth corresponds to CheckHostHealth for the plugin executable.
	CheckHostHealth(config backend.PluginConfig) *backend.CheckHealthResult

	// NewDataSourceInstance makes a request for a new datasource.
	NewDataSourceInstance(config backend.PluginConfig) (DataSourceInstance, error)
}

// DataSourceInstance implements each of the supported requests.  Alternativly this could be a set
// of interfaces that work for DataSource|AlertNotifier|etc|etc... with the support interrogated and
// maintaind by the helper on startup
type DataSourceInstance interface {
	CheckHealth() *backend.CheckHealthResult

	// If the request does not need access to the headers or user, use this request
	QueryData(req *backend.QueryDataRequest) (*backend.QueryDataResponse, error)

	// Get resource
	CallResource(req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error

	// Destroy lets you clean up any instance variables when the settings change
	Destroy()
}
