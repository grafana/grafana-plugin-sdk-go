package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
)

type ManageTestOpts struct {
	Address string
}

type TestPlugin struct {
	Client *TestPluginClient
	Server *TestPluginServer
}

func ManageForTest(instanceFactory datasource.InstanceFactoryFunc, opts ManageTestOpts) (TestPlugin, error) {
	handler := automanagement.NewManager(datasource.NewInstanceManager(instanceFactory))
	s, err := backend.TestStandaloneServe(backend.ServeOpts{
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		QueryDataHandler:    handler,
		StreamHandler:       handler,
	}, opts.Address)

	if err != nil {
		return TestPlugin{}, err
	}

	c, err := newTestPluginClient(opts.Address)
	if err != nil {
		return TestPlugin{}, err
	}

	return TestPlugin{
		Client: c,
		Server: newTestPluginServer(s),
	}, nil
}
