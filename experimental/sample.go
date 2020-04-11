package experimental

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

//----------------------------------------------------------------------------------
// PLUGIN HOST
//----------------------------------------------------------------------------------

// MyHost singelton host service
type MyHost struct{}

// CheckPluginHealth check if the plugin host is running
func (ds *MyDataSourceInstance) CheckPluginHealth(config backend.PluginConfig) backend.CheckHealthResult {
	return backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Plugin is running",
	}
}

// NewDataSourceInstance Create a new datasource instance
func (ds *MyDataSourceInstance) NewDataSourceInstance(config backend.PluginConfig) (*DataSourceInstance, error) {
	settings := myDataSourceConfig{
		url: config.DataSourceConfig.URL,
	}

	return &MyDataSourceInstance{
		config: settings,
		cache:  "TODO",
		client: "TODO",
	}, nil
}

//----------------------------------------------------------------------------------
// DATA SOURCE
//----------------------------------------------------------------------------------

type myDataSourceConfig struct {
	url  string
	port int32
}

// MyDataSourceInstance implements backend.DataSourceInstance
type MyDataSourceInstance struct {
	config myDataSourceConfig
	cache  interface{} // for example
	client interface{} // for example
}

// CheckHealth will check the currently configured settings
func (ds *MyDataSourceInstance) CheckHealth() backend.CheckHealthResult {
	if len(ds.config.url) < 2 {
		return backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "invalid URL",
		}
	}
	return backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Datasource is setup properly",
	}
}

// QueryData will run a set of queries
func (ds *MyDataSourceInstance) QueryData(req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// CallResource HTTP style reqource
func (ds *MyDataSourceInstance) CallResource(req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	sender.Send(&backend.CallResourceResponse{
		Status: 200,
		Headers: map[string][]string{
			"content-type": {"text/plain"},
		},
		Body: []byte("hello world"),
	})
	return nil
}

// Destroy destroy an instance (if necessary)
func (ds *MyDataSourceInstance) Destroy() bool {
	// TODO... can destroy
	return true
}
