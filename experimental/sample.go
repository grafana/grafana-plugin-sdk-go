package experimental

import (
	"fmt"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

//----------------------------------------------------------------------------------
// PLUGIN HOST
//----------------------------------------------------------------------------------

// MyHost singelton host service
type MyHost struct{}

// CheckHostHealth returns a backend.CheckHealthResult.
func (ds *MyHost) CheckHostHealth(config backend.PluginConfig) *backend.CheckHealthResult {
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Plugin is running",
	}
}

// NewDataSourceInstance Create a new datasource instance
func (ds *MyHost) NewDataSourceInstance(config backend.PluginConfig) (DataSourceInstance, error) {
	settings := myDataSourceSettings{
		url:  config.DataSourceConfig.URL,
		port: 1234,
	}

	return &MyDataSourceInstance{
		settings: settings,
		cache:    "TODO",
		client:   "TODO",
	}, nil
}

//----------------------------------------------------------------------------------
// DATA SOURCE instance
//----------------------------------------------------------------------------------

type myDataSourceSettings struct {
	url  string
	port int32
}

// MyDataSourceInstance implements backend.DataSourceInstance
type MyDataSourceInstance struct {
	settings myDataSourceSettings
	cache    interface{} // for example
	client   interface{} // for example
}

// CheckHealth will check the currently configured settings
func (ds *MyDataSourceInstance) CheckHealth() *backend.CheckHealthResult {
	if len(ds.settings.url) < 2 {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "invalid URL",
		}
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Datasource is setup properly",
	}
}

// QueryData will run a set of queries
func (ds *MyDataSourceInstance) QueryData(req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// CallResource HTTP style resource
func (ds *MyDataSourceInstance) CallResource(req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {

	if req.Path == "hello" {
		return SendJSON(sender, map[string]interface{}{"hello": "world"})
	}

	if req.Path == "text" {
		return SendPlainText(sender, "hello world")
	}

	return fmt.Errorf("unknown resource")
}

// Destroy destroy an instance (if necessary)
func (ds *MyDataSourceInstance) Destroy() {
	// If necessary, destroy the object (typically not required)
}

//----------------------------------------------------------------------------------
// SAMPLE MAIN
//----------------------------------------------------------------------------------

func MainXYZ() {
	// Setup the plugin environment
	backend.SetupPluginEnvironment("my-datasource")

	backend.Logger.Debug("Running my backend datasource")

	host := NewInstanceManager(&MyHost{})
	err := host.RunGRPCServer()
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
