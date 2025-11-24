package app_test

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type testApp struct {
	httpClient *http.Client
	backend.CallResourceHandler
}

func newApp(ctx context.Context, settings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	opts, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, err
	}

	client, err := httpclient.New(opts)
	if err != nil {
		return nil, err
	}

	app := &testApp{
		httpClient: client,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", app.handleTest)
	app.CallResourceHandler = httpadapter.New(mux)

	return app, nil
}

func (ds *testApp) Dispose() {
	// Cleanup
}

func (ds *testApp) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Handle request
	resp, err := ds.httpClient.Get("http://")
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()
	return nil, nil
}

func (ds *testApp) handleTest(rw http.ResponseWriter, _ *http.Request) {
	// Handle request
	resp, err := ds.httpClient.Get("http://")
	if err != nil {
		rw.WriteHeader(500)
		return
	}
	_ = resp.Body.Close()
}

func Example() {
	err := app.Manage("myapp-plugin-id", newApp, app.ManageOpts{})
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
