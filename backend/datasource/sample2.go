package datasource

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type myDataSourceInstance struct {
	httpClient *http.Client
	mux        *http.ServeMux
}

func newInstance(setting backend.DataSourceInstanceSettings) (Instance, error) {
	mux := http.NewServeMux()
	instance := &myDataSourceInstance{
		httpClient: &http.Client{},
		mux:        mux,
	}

	mux.Handle("/test", http.HandlerFunc(instance.handleTest))

	return instance, nil
}

func (ds *myDataSourceInstance) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Handle request
	_, _ = ds.httpClient.Get("http://")
	return nil, nil
}

func (ds *myDataSourceInstance) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return httpadapter.New(ds.mux).CallResource(ctx, req, sender)
}

func (ds *myDataSourceInstance) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// Handle request
	_, _ = ds.httpClient.Get("http://")
	return nil, nil
}

func (ds *myDataSourceInstance) handleTest(rw http.ResponseWriter, req *http.Request) {
	// Handle request
	_, _ = ds.httpClient.Get("http://")
}

func (ds *myDataSourceInstance) Dispose() {
	// Cleanup
}

func MainSample2() {
	ds := New(newInstance)
	err := ds.Serve()
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
