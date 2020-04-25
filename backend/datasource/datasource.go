package datasource

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// Instance defines the interface for data source plugin instances.
type Instance interface {
	backend.CheckHealthHandler
	backend.CallResourceHandler
	backend.QueryDataHandler
}

// InstanceProviderFunc is the factory method for creating a new data source plugin instance.
type InstanceProviderFunc func(settings backend.DataSourceInstanceSettings) (Instance, error)

type Plugin interface {
	ServeOpts() backend.ServeOpts
	Serve() error
}

type plugin struct {
	serveOpts backend.ServeOpts
	im        instancemgmt.InstanceManager
}

func New(fn InstanceProviderFunc) Plugin {
	if fn == nil {
		panic("fn cannot be nil")
	}

	ip := NewInstanceProvider(func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		instance, err := fn(settings)
		if err != nil {
			return nil, err
		}

		return instance, nil
	})

	p := &plugin{
		im: instancemgmt.New(ip),
	}
	serveOpts := backend.ServeOpts{
		CheckHealthHandler:  p,
		CallResourceHandler: p,
		QueryDataHandler:    p,
	}
	p.serveOpts = serveOpts

	return p
}

func (p *plugin) getInstance(pluginContext backend.PluginContext) (Instance, error) {
	if pluginContext.DataSourceInstanceSettings == nil {
		return nil, fmt.Errorf("data source instance settings cannot be nil")
	}

	instance, err := p.im.Get(pluginContext)
	if err != nil {
		return nil, err
	}

	return instance.(Instance), nil
}

func (p *plugin) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	instance, err := p.getInstance(req.PluginContext)
	if err != nil {
		return nil, err
	}

	return instance.CheckHealth(ctx, req)
}

func (p *plugin) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	instance, err := p.getInstance(req.PluginContext)
	if err != nil {
		return err
	}

	return instance.CallResource(ctx, req, sender)
}

func (p *plugin) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	instance, err := p.getInstance(req.PluginContext)
	if err != nil {
		return nil, err
	}

	return instance.QueryData(ctx, req)
}

func (p *plugin) ServeOpts() backend.ServeOpts {
	return p.serveOpts
}

func (p *plugin) Serve() error {
	return backend.Serve(p.serveOpts)
}
