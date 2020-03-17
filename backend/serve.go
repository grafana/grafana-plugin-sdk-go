package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/plugin"
	"github.com/prometheus/client_golang/prometheus"
)

//ServeOpts options for serving plugins.
type ServeOpts struct {
	CheckHealthHandler   CheckHealthHandler
	CallResourceHandler  CallResourceHandler
	QueryDataHandler     QueryDataHandler
	TransformDataHandler TransformDataHandler
}

type ConfigurePlugin struct {
	Metrics prometheus.Registerer
}

type ServePluginFunc func(logger log.Logger, c ConfigurePlugin) ServeOpts

// Serve starts serving the plugin over gRPC.
func Serve(fn ServePluginFunc) {
	logger := log.New()
	c := ConfigurePlugin{
		Metrics: prometheus.DefaultRegisterer,
	}
	opts := fn(logger, c)

	sdkAdapter := &sdkAdapter{
		metricGatherer:       prometheus.DefaultGatherer,
		CheckHealthHandler:   opts.CheckHealthHandler,
		CallResourceHandler:  opts.CallResourceHandler,
		QueryDataHandler:     opts.QueryDataHandler,
		TransformDataHandler: opts.TransformDataHandler,
	}

	pluginOpts := plugin.ServeOpts{
		DiagnosticsServer: sdkAdapter,
	}

	if opts.CallResourceHandler != nil {
		pluginOpts.ResourceServer = sdkAdapter
	}

	if opts.QueryDataHandler != nil {
		pluginOpts.DataServer = sdkAdapter
	}

	if opts.TransformDataHandler != nil {
		pluginOpts.TransformServer = sdkAdapter
	}

	plugin.Serve(pluginOpts)
}

type Plugin interface {
	CheckHealthHandler
	CallResourceHandler
}

type DataSourcePlugin interface {
	Plugin
	QueryDataHandler
}

type TransformPlugin interface {
	TransformDataHandler
}

type PluginFactoryFunc func(logger log.Logger, c ConfigurePlugin) Plugin
type DataSourcePluginFactoryFunc func(logger log.Logger, c ConfigurePlugin) DataSourcePlugin
type TransformPluginFactoryFunc func(logger log.Logger, c ConfigurePlugin) TransformPlugin

//ServePluginOpts options for serving plugins.
type ServePluginOpts struct {
	PluginProvider           PluginFactoryFunc
	DataSourcePluginProvider DataSourcePluginFactoryFunc
	TransformPluginProvider  TransformPluginFactoryFunc
}

func servePluginExample(opts ServePluginOpts) {
	logger := log.New()
	c := ConfigurePlugin{
		Metrics: prometheus.DefaultRegisterer,
	}

	sdkAdapter := &sdkAdapter{
		metricGatherer: prometheus.DefaultGatherer,
	}

	if opts.PluginProvider != nil {
		p := opts.PluginProvider(logger, c)
		sdkAdapter.CheckHealthHandler = p
		sdkAdapter.CallResourceHandler = p

		plugin.Serve(plugin.ServeOpts{
			DiagnosticsServer: sdkAdapter,
			ResourceServer:    sdkAdapter,
		})
		return
	}

	if opts.DataSourcePluginProvider != nil {
		p := opts.DataSourcePluginProvider(logger, c)
		sdkAdapter.CheckHealthHandler = p
		sdkAdapter.CallResourceHandler = p
		sdkAdapter.QueryDataHandler = p

		plugin.Serve(plugin.ServeOpts{
			DiagnosticsServer: sdkAdapter,
			ResourceServer:    sdkAdapter,
			DataServer:        sdkAdapter,
		})
		return
	}

	if opts.TransformPluginProvider != nil {
		p := opts.TransformPluginProvider(logger, c)
		sdkAdapter.TransformDataHandler = p

		plugin.Serve(plugin.ServeOpts{
			TransformServer: sdkAdapter,
		})
		return
	}

	panic("invalid arguments for serve plugin")
}

// ServePlugin starts serving the plugin over gRPC.
func ServePlugin(factory PluginFactoryFunc) {
	if factory == nil {
		panic("factory func cannot be nil")
	}

	servePluginExample(ServePluginOpts{
		PluginProvider: factory,
	})
}

// ServeDataSourcePlugin starts serving the data source plugin over gRPC.
func ServeDataSourcePlugin(factory DataSourcePluginFactoryFunc) {
	if factory == nil {
		panic("factory func cannot be nil")
	}

	servePluginExample(ServePluginOpts{
		DataSourcePluginProvider: factory,
	})
}

// ServeTransformPlugin starts serving the plugin over gRPC.
func ServeTransformPlugin(factory TransformPluginFactoryFunc) {
	if factory == nil {
		panic("factory func cannot be nil")
	}

	servePluginExample(ServePluginOpts{
		TransformPluginProvider: factory,
	})
}
