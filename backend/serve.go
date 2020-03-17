package backend

import (
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

// Serve starts serving the plugin over gRPC.
func Serve(opts ServeOpts) error {
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

	return plugin.Serve(pluginOpts)
}
