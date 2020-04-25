package app

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type myAppInstance struct {
	httpClient *http.Client
	mux        *http.ServeMux
}

func newInstance(setting backend.AppInstanceSettings) (Instance, error) {
	mux := http.NewServeMux()
	instance := &myAppInstance{
		httpClient: &http.Client{},
		mux:        mux,
	}

	mux.Handle("/test", http.HandlerFunc(instance.handleTest))

	return instance, nil
}

func (a *myAppInstance) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Handle request
	_, _ = a.httpClient.Get("http://")
	return nil, nil
}

func (a *myAppInstance) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return httpadapter.New(a.mux).CallResource(ctx, req, sender)
}

func (a *myAppInstance) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// Handle request
	_, _ = a.httpClient.Get("http://")
	return nil, nil
}

func (a *myAppInstance) handleTest(rw http.ResponseWriter, req *http.Request) {
	// Handle request
	_, _ = a.httpClient.Get("http://")
}

func (a *myAppInstance) Dispose() {
	// Cleanup
}

func MainSample2() {
	a := New(newInstance)
	err := a.Serve()
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
