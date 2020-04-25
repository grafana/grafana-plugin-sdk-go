package app

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

type myAppInstanceSettings struct {
	httpClient *http.Client
}

func newInstanceSettings(setting backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	return &myAppInstanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *myAppInstanceSettings) Dispose() {
	// Cleanup
}

type myApp struct {
	im instancemgmt.InstanceManager
}

func newApp() backend.ServeOpts {
	ip := NewInstanceProvider(newInstanceSettings)
	a := &myApp{
		im: instancemgmt.New(ip),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", a.handleTest)

	return backend.ServeOpts{
		CheckHealthHandler:  a,
		CallResourceHandler: httpadapter.New(mux),
	}
}

func (a *myApp) getSettings(pluginContext backend.PluginContext) (*myAppInstanceSettings, error) {
	iface, err := a.im.Get(pluginContext)
	if err != nil {
		return nil, err
	}

	return iface.(*myAppInstanceSettings), nil
}

func (a *myApp) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	settings, err := a.getSettings(req.PluginContext)
	if err != nil {
		return nil, err
	}

	// Handle request
	_, _ = settings.httpClient.Get("http://")
	return nil, nil
}

func (a *myApp) handleTest(rw http.ResponseWriter, req *http.Request) {
	pluginContext := httpadapter.PluginConfigFromContext(req.Context())
	settings, err := a.getSettings(pluginContext)
	if err != nil {
		rw.WriteHeader(500)
		return
	}

	// Handle request
	_, _ = settings.httpClient.Get("http://")
}

func MainSample() {
	err := backend.Serve(newApp())
	if err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
