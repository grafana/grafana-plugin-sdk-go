package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type TransformPlugin interface {
	CorePlugin
	TransformQueryDataHandler
}

// TransformImpl implements the plugin interface from github.com/hashicorp/go-plugin.
type TransformImpl struct {
	plugin.NetRPCUnsupportedPlugin
	backend   backendWrapper
	transform transformWrapper
}

func (t *TransformImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterBackendServer(s, &transformGRPCServer{
		backend:   t.backend,
		transform: t.transform,
		broker:    broker,
	})
	return nil
}

func (t *TransformImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &transformGRPCClient{client: pluginv2.NewBackendClient(c), broker: broker}, nil
}

// Callback

type TransformCallBack interface {
	QueryData(ctx context.Context, req *pluginv2.QueryData_Request) (*pluginv2.QueryData_Response, error)
}
