package backend

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// BackendGRPCPlugin implements the plugin interface from github.com/hashicorp/go-plugin.
type BackendGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Wrapper backendWrapper
}

func (t *BackendGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterBackendServer(s, &backendGRPCServer{
		impl: t.Wrapper,
	})
	return nil
}

func (t *BackendGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &transformGRPCClient{client: pluginv2.NewBackendClient(c), broker: broker}, nil
}

type backendGRPCServer struct {
	impl backendWrapper
}

func (s *backendGRPCServer) GetSchema(ctx context.Context, req *pluginv2.GetSchema_Request) (*pluginv2.GetSchema_Response, error) {
	return nil, nil
}

func (s *backendGRPCServer) ValidatePluginConfig(ctx context.Context, req *pluginv2.ValidatePluginConfig_Request) (*pluginv2.ValidatePluginConfig_Response, error) {
	return nil, nil
}

func (s *backendGRPCServer) Configure(ctx context.Context, req *pluginv2.Configure_Request) (*pluginv2.Configure_Response, error) {
	return nil, nil
}

func (s *backendGRPCServer) CallResource(ctx context.Context, req *pluginv2.CallResource_Request) (*pluginv2.CallResource_Response, error) {
	return nil, nil
}

func (s *backendGRPCServer) QueryData(ctx context.Context, req *pluginv2.QueryData_Request) (*pluginv2.QueryData_Response, error) {
	return nil, nil
}

func (s *backendGRPCServer) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	return nil, nil
}

func (s *backendGRPCServer) CheckHealth(ctx context.Context, req *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	return nil, nil
}

type backendGRPCClient struct {
	pluginv2.BackendServer
	client pluginv2.BackendClient
}

func (c *backendGRPCClient) GetSchema(ctx context.Context, req *pluginv2.GetSchema_Request) (*pluginv2.GetSchema_Response, error) {
	return c.client.GetSchema(ctx, req)
}

func (c *backendGRPCClient) ValidatePluginConfig(ctx context.Context, req *pluginv2.ValidatePluginConfig_Request) (*pluginv2.ValidatePluginConfig_Response, error) {
	return c.client.ValidatePluginConfig(ctx, req)
}

func (c *backendGRPCClient) Configure(ctx context.Context, req *pluginv2.Configure_Request) (*pluginv2.Configure_Response, error) {
	return c.client.Configure(ctx, req)
}

func (c *backendGRPCClient) CallResource(ctx context.Context, req *pluginv2.CallResource_Request) (*pluginv2.CallResource_Response, error) {
	return c.client.CallResource(ctx, req)
}

func (c *backendGRPCClient) QueryData(ctx context.Context, req *pluginv2.QueryData_Request) (*pluginv2.QueryData_Response, error) {
	return c.client.QueryData(ctx, req)
}

func (c *backendGRPCClient) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	return c.client.CollectMetrics(ctx, req)
}

func (c *backendGRPCClient) CheckHealth(ctx context.Context, req *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	return c.client.CheckHealth(ctx, req)
}
