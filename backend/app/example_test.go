package app_test

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type testAppInstanceSettings struct {
	httpClient *http.Client
}

func newAppInstance(setting backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	return &testAppInstanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *testAppInstanceSettings) Dispose() {
	// Cleanup
}

type testApp struct {
	im instancemgmt.InstanceManager
}

func newApp(im instancemgmt.InstanceManager) app.ServeOpts {
	a := &testApp{
		im: im,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", a.handleTest)

	return app.ServeOpts{
		CheckHealthHandler:  a,
		CallResourceHandler: httpadapter.New(mux),
	}
}

func (a *testApp) getSettings(pluginContext backend.PluginContext) (*testAppInstanceSettings, error) {
	iface, err := a.im.Get(pluginContext)
	if err != nil {
		return nil, err
	}

	return iface.(*testAppInstanceSettings), nil
}

func (a *testApp) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	settings, err := a.getSettings(req.PluginContext)
	if err != nil {
		return nil, err
	}

	// Handle request
	_, _ = settings.httpClient.Get("http://")
	return nil, nil
}

func (a *testApp) handleTest(rw http.ResponseWriter, req *http.Request) {
	pluginContext := httpadapter.PluginConfigFromContext(req.Context())
	settings, err := a.getSettings(pluginContext)
	if err != nil {
		rw.WriteHeader(500)
		return
	}

	// Handle request
	_, _ = settings.httpClient.Get("http://")
}

func Example() {
	p := app.New(newAppInstance, newApp)
	err := p.Serve()
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
