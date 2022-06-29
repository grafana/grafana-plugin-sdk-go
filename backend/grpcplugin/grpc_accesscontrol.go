package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// AccessControlServer represents a data server.
type AccessControlServer interface {
	pluginv2.AccessControlServer
}

// AccessControlClient represents a data client.
type AccessControlClient interface {
	pluginv2.AccessControlClient
}

// AccessControlGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type AccessControlGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	AccessControlServer AccessControlServer
}

// GRPCServer registers p as a data gRPC server.
func (p *AccessControlGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterAccessControlServer(s, &accessControlGRPCServer{
		server: p.AccessControlServer,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *AccessControlGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &accessControlGRPCClient{client: pluginv2.NewAccessControlClient(c)}, nil
}

type accessControlGRPCServer struct {
	server AccessControlServer
}

// HasAccess queries s for access control.
func (s *accessControlGRPCServer) HasAccess(ctx context.Context, req *pluginv2.HasAccessRequest) (*pluginv2.HasAccessResponse, error) {
	return s.server.HasAccess(ctx, req)
}

type accessControlGRPCClient struct {
	client pluginv2.AccessControlClient
}

// HasAccess queries m for access control.
func (m *accessControlGRPCClient) HasAccess(ctx context.Context, req *pluginv2.HasAccessRequest, opts ...grpc.CallOption) (*pluginv2.HasAccessResponse, error) {
	return m.client.HasAccess(ctx, req, opts...)
}

var _ AccessControlServer = &accessControlGRPCServer{}
var _ AccessControlClient = &accessControlGRPCClient{}
