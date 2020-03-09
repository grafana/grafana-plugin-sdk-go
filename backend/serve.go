package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/plugin"
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
	sdkAdapter := &sdkAdapter{
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

	return plugin.Serve(pluginOpts)
}
