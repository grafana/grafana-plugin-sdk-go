package main

import (
	"context"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
)

type fixtureApp struct{}

func (fixtureApp) CallResource(ctx context.Context, _ *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return sender.Send(&backend.CallResourceResponse{Status: http.StatusOK, Body: []byte("app fixture resource")})
}

func (fixtureApp) CheckHealth(context.Context, *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk, Message: "app fixture healthy"}, nil
}

func newApp(context.Context, backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	return fixtureApp{}, nil
}

func main() {
	if err := app.Manage("marketplace-app-fixture", newApp, app.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{}),
	}); err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
