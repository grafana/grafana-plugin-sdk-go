package datasource

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type myDataSourceInstanceSettings struct {
	httpClient *http.Client
}

func newInstanceSettings(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &myDataSourceInstanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *myDataSourceInstanceSettings) Dispose() {
	// Cleanup
}

type myDataSource struct {
	im instancemgmt.InstanceManager
}

func newDataSource() backend.ServeOpts {
	ip := NewInstanceProvider(newInstanceSettings)
	ds := &myDataSource{
		im: instancemgmt.New(ip),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", ds.handleTest)

	return backend.ServeOpts{
		CheckHealthHandler:  ds,
		CallResourceHandler: httpadapter.New(mux),
		QueryDataHandler:    ds,
	}
}

func (ds *myDataSource) getSettings(pluginContext backend.PluginContext) (*myDataSourceInstanceSettings, error) {
	iface, err := ds.im.Get(pluginContext)
	if err != nil {
		return nil, err
	}

	return iface.(*myDataSourceInstanceSettings), nil
}

func (ds *myDataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	settings, err := ds.getSettings(req.PluginContext)
	if err != nil {
		return nil, err
	}

	// Handle request
	_, _ = settings.httpClient.Get("http://")
	return nil, nil
}

func (ds *myDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	var resp *backend.QueryDataResponse
	err := ds.im.Do(req.PluginContext, func(settings *myDataSourceInstanceSettings) error {
		// Handle request
		_, _ = settings.httpClient.Get("http://")
		return nil
	})

	return resp, err
}

func (ds *myDataSource) handleTest(rw http.ResponseWriter, req *http.Request) {
	pluginContext := httpadapter.PluginConfigFromContext(req.Context())
	settings, err := ds.getSettings(pluginContext)
	if err != nil {
		rw.WriteHeader(500)
		return
	}

	// Handle request
	_, _ = settings.httpClient.Get("http://")
}

func MainSample() {
	err := backend.Serve(newDataSource())
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
