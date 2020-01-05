package backend

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// PluginImpl implements the plugin interface from github.com/hashicorp/go-plugin.
type CoreImpl struct {
	plugin.NetRPCUnsupportedPlugin

	Wrap coreWrapper
}

func (p *CoreImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// pluginv2.RegisterCoreServer(s, &coreGRPCServer{
	// 	Impl: p.Wrap,
	// })
	return nil
}

func (p *CoreImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// return &CoreGRPCClient{client: pluginv2.NewCoreClient(c)}, nil
	return nil, nil
}
