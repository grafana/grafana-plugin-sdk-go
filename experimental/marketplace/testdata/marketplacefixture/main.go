package main

import (
	"context"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
)

func newDataSource(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return struct{}{}, nil
}

func main() {
	if err := marketplace.Manage("marketplace-fixture", newDataSource, datasource.ManageOpts{}); err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
