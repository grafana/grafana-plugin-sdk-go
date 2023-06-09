package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// MetadataServer represents a metadata server.
type MetadataServer interface {
	pluginv2.MetadataServer
}

// MetadataClient represents a data client.
type MetadataClient interface {
	pluginv2.MetadataClient
}

// MetadataGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type MetadataGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	MetadataServer MetadataServer
}

// GRPCServer registers p as a data gRPC server.
func (p *MetadataGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterMetadataServer(s, &metadataGRPCServer{
		server: p.MetadataServer,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *MetadataGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &metadataGRPCClient{client: pluginv2.NewMetadataClient(c)}, nil
}

type metadataGRPCServer struct {
	server MetadataServer
}

// ProvideMetadata queries s for metadata.
func (s *metadataGRPCServer) ProvideMetadata(ctx context.Context, req *pluginv2.ProvideMetadataRequest) (*pluginv2.ProvideMetadataResponse, error) {
	return s.server.ProvideMetadata(ctx, req)
}

type metadataGRPCClient struct {
	client pluginv2.MetadataClient
}

// ProvideMetadata queries m for metadata.
func (m *metadataGRPCClient) ProvideMetadata(ctx context.Context, req *pluginv2.ProvideMetadataRequest, opts ...grpc.CallOption) (*pluginv2.ProvideMetadataResponse, error) {
	return m.client.ProvideMetadata(ctx, req, opts...)
}

var _ MetadataServer = &metadataGRPCServer{}
var _ MetadataClient = &metadataGRPCClient{}
