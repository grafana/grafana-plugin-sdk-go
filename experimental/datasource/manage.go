package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
	"github.com/grafana/grafana-plugin-sdk-go/internal/standalone"
)

type ManageTestOpts struct {
	Address string
	Dir     string
}

func ManageForTest(instanceFactory datasource.InstanceFactoryFunc, opts ManageTestOpts) error {
	handler := automanagement.NewManager(datasource.NewInstanceManager(instanceFactory))
	return backend.GracefulStandaloneServe(backend.ServeOpts{
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		QueryDataHandler:    handler,
		StreamHandler:       handler,
	}, standalone.NewServerSettings(opts.Address, opts.Dir))
}
