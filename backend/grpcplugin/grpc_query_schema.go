package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// QuerySchemaServer represents a query schema server.
type QuerySchemaServer interface {
	pluginv2.QuerySchemaServer
}

// QuerySchemaClient represents a query schema client.
type QuerySchemaClient interface {
	pluginv2.QuerySchemaClient
}

// QuerySchemaGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type QuerySchemaGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	QuerySchemaServer QuerySchemaServer
}

// GRPCServer registers p as a query schema gRPC server.
func (p *QuerySchemaGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterQuerySchemaServer(s, &querySchemaGRPCServer{
		server: p.QuerySchemaServer,
	})
	return nil
}

// GRPCClient returns c as a query schema gRPC client.
func (p *QuerySchemaGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &querySchemaGRPCClient{client: pluginv2.NewQuerySchemaClient(c)}, nil
}

type querySchemaGRPCServer struct {
	server QuerySchemaServer
}

func (s *querySchemaGRPCServer) GetQuerySchema(ctx context.Context, req *pluginv2.GetQuerySchemaRequest) (*pluginv2.GetQuerySchemaResponse, error) {
	return s.server.GetQuerySchema(ctx, req)
}

type querySchemaGRPCClient struct {
	client pluginv2.QuerySchemaClient
}

func (c *querySchemaGRPCClient) GetQuerySchema(ctx context.Context, req *pluginv2.GetQuerySchemaRequest, opts ...grpc.CallOption) (*pluginv2.GetQuerySchemaResponse, error) {
	return c.client.GetQuerySchema(ctx, req, opts...)
}

var _ QuerySchemaServer = &querySchemaGRPCServer{}
var _ QuerySchemaClient = &querySchemaGRPCClient{}
