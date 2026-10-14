package main

import (
	"context"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
)

type fixtureDataSource struct{}

func (fixtureDataSource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	resp := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		resp.Responses[q.RefID] = backend.DataResponse{}
	}
	return resp, nil
}

func (fixtureDataSource) CheckHealth(context.Context, *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk, Message: "datasource fixture healthy"}, nil
}

func newDataSource(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return fixtureDataSource{}, nil
}

func main() {
	if err := datasource.Manage("marketplace-fixture", newDataSource, datasource.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{}),
	}); err != nil {
		backend.Logger.Error(err.Error())
		os.Exit(1)
	}
}
