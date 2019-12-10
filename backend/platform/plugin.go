package platform

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// PlatformImpl implements the plugin interface from github.com/hashicorp/go-plugin.
type PlatformImpl struct {
	plugin.NetRPCUnsupportedPlugin

	Wrap PlatformPluginWrapper
}

func (p *PlatformImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	bproto.RegisterGrafanaPlatformServer(s, &GRPCServer{
		Impl:   p.Wrap,
		broker: broker,
	})
	return nil
}

func (p *PlatformImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: bproto.NewGrafanaPlatformClient(c)}, nil
}
