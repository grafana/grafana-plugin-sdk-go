package datasource_test

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type testDataSource struct {
	httpClient *http.Client
	backend.CallResourceHandler
}

func newDataSource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	opts, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, err
	}

	client, err := httpclient.New(opts)
	if err != nil {
		return nil, err
	}

	ds := &testDataSource{
		httpClient: client,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", ds.handleTest)
	ds.CallResourceHandler = httpadapter.New(mux)

	return ds, nil
}

func (ds *testDataSource) Dispose() {
	// Cleanup
}

func (ds *testDataSource) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Handle request
	resp, err := ds.httpClient.Get("http://")
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()
	return nil, nil
}

func (ds *testDataSource) QueryData(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	var resp *backend.QueryDataResponse
	// Handle request
	httpResp, err := ds.httpClient.Get("http://")
	if err != nil {
		return nil, err
	}
	_ = httpResp.Body.Close()

	return resp, err
}

func (ds *testDataSource) handleTest(rw http.ResponseWriter, _ *http.Request) {
	// Handle request
	resp, err := ds.httpClient.Get("http://")
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	_ = resp.Body.Close()
}

func Example() {
	err := datasource.Manage("myds-plugin-id", newDataSource, datasource.ManageOpts{})
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
