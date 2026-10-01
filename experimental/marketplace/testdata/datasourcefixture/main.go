package main

import (
	"context"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

func newDataSource(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return struct{}{}, nil
}

func main() {
	if err := datasource.Manage("datasource-fixture", newDataSource, datasource.ManageOpts{}); err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
