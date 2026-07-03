package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// StoredObjectEventsServer represents a stored object events server.
type StoredObjectEventsServer interface {
	pluginv2.StoredObjectEventsServer
}

// StoredObjectEventsClient represents a stored object events client.
type StoredObjectEventsClient interface {
	pluginv2.StoredObjectEventsClient
}

// StoredObjectEventsGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type StoredObjectEventsGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	StoredObjectEventsServer StoredObjectEventsServer
}

// GRPCServer registers p as a stored object events gRPC server.
func (p *StoredObjectEventsGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterStoredObjectEventsServer(s, &storedObjectEventsGRPCServer{
		server: p.StoredObjectEventsServer,
	})
	return nil
}

func (p *StoredObjectEventsGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &storedObjectEventsGRPCClient{client: pluginv2.NewStoredObjectEventsClient(c)}, nil
}

type storedObjectEventsGRPCServer struct {
	server StoredObjectEventsServer
}

func (s *storedObjectEventsGRPCServer) StreamStoredObjectEvents(stream grpc.ClientStreamingServer[pluginv2.StoredObjectEvent, pluginv2.StoredObjectEventsResponse]) error {
	return s.server.StreamStoredObjectEvents(stream)
}

type storedObjectEventsGRPCClient struct {
	client StoredObjectEventsClient
}

func (s *storedObjectEventsGRPCClient) StreamStoredObjectEvents(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[pluginv2.StoredObjectEvent, pluginv2.StoredObjectEventsResponse], error) {
	return s.client.StreamStoredObjectEvents(ctx, opts...)
}

var _ StoredObjectEventsServer = &storedObjectEventsGRPCServer{}
var _ StoredObjectEventsClient = &storedObjectEventsGRPCClient{}
