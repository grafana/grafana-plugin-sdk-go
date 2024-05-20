package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// InstanceSettingsServer represents a data server.
type InstanceSettingsServer interface {
	pluginv2.InstanceSettingsServer
}

// InstanceSettingsClient represents a data client.
type InstanceSettingsClient interface {
	pluginv2.InstanceSettingsClient
}

// InstanceSettingsGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type InstanceSettingsGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	InstanceSettingsServer InstanceSettingsServer
}

// GRPCServer registers p as a data gRPC server.
func (p *InstanceSettingsGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterInstanceSettingsServer(s, &instanceSettingsGRPCServer{
		server: p.InstanceSettingsServer,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *InstanceSettingsGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &instanceSettingsGRPCClient{client: pluginv2.NewInstanceSettingsClient(c)}, nil
}

type instanceSettingsGRPCServer struct {
	server InstanceSettingsServer
}

func (s *instanceSettingsGRPCServer) CreateInstanceSettings(ctx context.Context, req *pluginv2.CreateInstanceSettingsRequest) (*pluginv2.InstanceSettingsResponse, error) {
	return s.server.CreateInstanceSettings(ctx, req)
}

func (s *instanceSettingsGRPCServer) UpdateInstanceSettings(ctx context.Context, req *pluginv2.UpdateInstanceSettingsRequest) (*pluginv2.InstanceSettingsResponse, error) {
	return s.server.UpdateInstanceSettings(ctx, req)
}

type instanceSettingsGRPCClient struct {
	client pluginv2.InstanceSettingsClient
}

func (c *instanceSettingsGRPCClient) CreateInstanceSettings(ctx context.Context, req *pluginv2.CreateInstanceSettingsRequest, opts ...grpc.CallOption) (*pluginv2.InstanceSettingsResponse, error) {
	return c.client.CreateInstanceSettings(ctx, req, opts...)
}

func (c *instanceSettingsGRPCClient) UpdateInstanceSettings(ctx context.Context, req *pluginv2.UpdateInstanceSettingsRequest, opts ...grpc.CallOption) (*pluginv2.InstanceSettingsResponse, error) {
	return c.client.UpdateInstanceSettings(ctx, req, opts...)
}

var _ InstanceSettingsServer = &instanceSettingsGRPCServer{}
var _ InstanceSettingsClient = &instanceSettingsGRPCClient{}
