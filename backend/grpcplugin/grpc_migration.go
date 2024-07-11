package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// QueryMigrationServer represents an migration control server.
type QueryMigrationServer interface {
	pluginv2.QueryMigrationControlServer
}

// QueryMigrationClient represents an migration control client.
type QueryMigrationClient interface {
	pluginv2.QueryMigrationControlClient
}

// MigrationGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type MigrationGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	MigrationServer QueryMigrationServer
}

// GRPCServer registers p as an Migration control gRPC server.
func (p *MigrationGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterQueryMigrationControlServer(s, &migrationGRPCServer{
		server: p.MigrationServer,
	})
	return nil
}

func (p *MigrationGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &migrationGRPCClient{client: pluginv2.NewQueryMigrationControlClient(c)}, nil
}

type migrationGRPCServer struct {
	server QueryMigrationServer
}

func (s *migrationGRPCServer) MigrateQuery(ctx context.Context, req *pluginv2.QueryMigrationRequest) (*pluginv2.QueryMigrationResponse, error) {
	return s.server.MigrateQuery(ctx, req)
}

type migrationGRPCClient struct {
	client pluginv2.QueryMigrationControlClient
}

func (s *migrationGRPCClient) MigrateQuery(ctx context.Context, req *pluginv2.QueryMigrationRequest, opts ...grpc.CallOption) (*pluginv2.QueryMigrationResponse, error) {
	return s.client.MigrateQuery(ctx, req, opts...)
}

var _ QueryMigrationServer = &migrationGRPCServer{}
var _ QueryMigrationClient = &migrationGRPCClient{}
