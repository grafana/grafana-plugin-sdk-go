package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// RegistrationServer represents a registration server.
type RegistrationServer interface {
	pluginv2.RegistrationServer
}

// RegistrationClient represents a registration client.
type RegistrationClient interface {
	pluginv2.RegistrationClient
}

// RegistrationGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type RegistrationGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	RegistrationServer RegistrationServer
}

// GRPCServer registers p as a registration gRPC server.
func (p *RegistrationGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterRegistrationServer(s, &registrationGRPCServer{
		server: p.RegistrationServer,
	})
	return nil
}

// GRPCClient returns c as a registration gRPC client.
func (p *RegistrationGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &registrationGRPCClient{client: pluginv2.NewRegistrationClient(c)}, nil
}

type registrationGRPCServer struct {
	server RegistrationServer
}

// QueryRoles queries s for role registrations.
func (s *registrationGRPCServer) QueryRoles(ctx context.Context, req *pluginv2.QueryRolesRequest) (*pluginv2.QueryRolesResponse, error) {
	return s.server.QueryRoles(ctx, req)
}

type registrationGRPCClient struct {
	client RegistrationClient
}

// QueryRoles queries m for role registrations.
func (m *registrationGRPCClient) QueryRoles(ctx context.Context, req *pluginv2.QueryRolesRequest, opts ...grpc.CallOption) (*pluginv2.QueryRolesResponse, error) {
	return m.client.QueryRoles(ctx, req, opts...)
}

var _ RegistrationServer = &registrationGRPCServer{}
var _ RegistrationClient = &registrationGRPCClient{}
