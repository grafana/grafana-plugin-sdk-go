package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Plugin interface {
	CheckDataSourceHealthHandler
	CallDataSourceResourceHandler
	backend.QueryDataHandler
}

type PluginFactoryFunc func(logger log.Logger, c backend.ConfigurePlugin) Plugin

func Serve(fn PluginFactoryFunc) {
	backend.Serve(func(logger log.Logger, c backend.ConfigurePlugin) backend.ServeOpts {
		ds := fn(logger, c)
		return backend.ServeOpts{
			CheckHealthHandler:  CheckDataSourceHealthHandlerFunc(ds.CheckDataSourceHealth),
			CallResourceHandler: CallDataSourceResourceHandlerFunc(ds.CallDataSourceResource),
			QueryDataHandler:    ds,
		}
	})
}
