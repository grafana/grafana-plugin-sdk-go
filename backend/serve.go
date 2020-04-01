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

	pluginOpts := plugin.ServeOpts{
		DiagnosticsServer: newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, opts.CheckHealthHandler),
	}

	if opts.CallResourceHandler != nil {
		pluginOpts.ResourceServer = newResourceSDKAdapter(opts.CallResourceHandler)
	}

	if opts.QueryDataHandler != nil {
		pluginOpts.DataServer = newDataSDKAdapter(opts.QueryDataHandler)
	}

	if opts.TransformDataHandler != nil {
		pluginOpts.TransformServer = newTransformSDKAdapter(opts.TransformDataHandler)
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

func ServePluginExample(opts ServePluginOpts) {
	logger := log.New()
	c := ConfigurePlugin{
		Metrics: prometheus.DefaultRegisterer,
	}

	if opts.PluginProvider != nil {
		p := opts.PluginProvider(logger, c)
		plugin.Serve(plugin.ServeOpts{
			DiagnosticsServer: newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, p),
			ResourceServer:    newResourceSDKAdapter(p),
		})
		return
	}

	if opts.DataSourcePluginProvider != nil {
		p := opts.DataSourcePluginProvider(logger, c)
		plugin.Serve(plugin.ServeOpts{
			DiagnosticsServer: newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, p),
			ResourceServer:    newResourceSDKAdapter(p),
			DataServer:        newDataSDKAdapter(p),
		})
		return
	}

	if opts.TransformPluginProvider != nil {
		p := opts.TransformPluginProvider(logger, c)
		plugin.Serve(plugin.ServeOpts{
			TransformServer: newTransformSDKAdapter(p),
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

	ServePluginExample(ServePluginOpts{
		PluginProvider: factory,
	})
}

// ServeDataSourcePlugin starts serving the data source plugin over gRPC.
func ServeDataSourcePlugin(factory DataSourcePluginFactoryFunc) {
	if factory == nil {
		panic("factory func cannot be nil")
	}

	ServePluginExample(ServePluginOpts{
		DataSourcePluginProvider: factory,
	})
}

// ServeTransformPlugin starts serving the plugin over gRPC.
func ServeTransformPlugin(factory TransformPluginFactoryFunc) {
	if factory == nil {
		panic("factory func cannot be nil")
	}

	ServePluginExample(ServePluginOpts{
		TransformPluginProvider: factory,
	})
}
