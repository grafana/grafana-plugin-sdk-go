package main

import (
	"context"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
)

func newApp(context.Context, backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	return struct{}{}, nil
}

func main() {
	if err := app.Manage("marketplace-app-fixture", newApp, app.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{}),
	}); err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
