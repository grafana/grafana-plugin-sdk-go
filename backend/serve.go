package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/common"
	plugin "github.com/hashicorp/go-plugin"
)

type ServeOpts struct {
	BackendProvider    func(plugin ConfigurePlugin)
	DatasourceProvider func(plugin ConfigurePlugin) QueryDataHandler
	TransformProvider  func(plugin ConfigurePlugin) TransformQueryDataHandler
}

// Serve starts serving the datasource plugin over gRPC.
func Serve(opts ServeOpts) error {
	versionedPlugins := make(map[int]plugin.PluginSet)
	pSet := make(plugin.PluginSet)
	var p *PluginImpl

	if opts.BackendProvider != nil {
		builder := newBackendPluginConfigurer()
		opts.BackendProvider(builder)
		p = builder.build()
	}

	if opts.DatasourceProvider != nil {
		builder := newDatasourcePluginConfigurer()
		opts.DatasourceProvider(builder)
		p = builder.build()
	}

	if opts.TransformProvider != nil {
		builder := newBackendPluginConfigurer()
		opts.TransformProvider(builder)
		p = builder.build()
	}

	if p.diagnostics != nil {
		pSet["diagnostics"] = p.diagnostics
	}

	if p.backend != nil {
		pSet["backend"] = p.backend
	}

	if p.datasource != nil {
		pSet["datasource"] = p.datasource
	}

	if p.transform != nil {
		pSet["transform"] = p.transform
	}

	versionedPlugins[common.ProtocolVersion] = pSet

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  common.Handshake,
		VersionedPlugins: versionedPlugins,
		GRPCServer:       plugin.DefaultGRPCServer,
	})

	return nil
}
