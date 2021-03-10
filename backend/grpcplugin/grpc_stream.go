package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// StreamServer represents a stream server.
type StreamServer interface {
	pluginv2.StreamServer
}

// StreamClient represents a stream client.
type StreamClient interface {
	pluginv2.StreamClient
}

// StreamGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type StreamGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	StreamServer StreamServer
}

// GRPCServer registers p as a resource gRPC server.
func (p *StreamGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterStreamServer(s, &streamGRPCServer{
		server: p.StreamServer,
	})
	return nil
}

// GRPCClient returns c as a resource gRPC client.
func (p *StreamGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &streamGRPCClient{client: pluginv2.NewStreamClient(c)}, nil
}

type streamGRPCServer struct {
	server StreamServer
}

//func (s streamGRPCServer) InitStreams(request *pluginv2.StreamInitRequest, server pluginv2.Stream_InitStreamsServer) error {
//	return s.server.InitStreams(request, server)
//}
//
//func (s streamGRPCServer) GetStreamCapabilities(ctx context.Context, request *pluginv2.StreamCapabilitiesRequest) (*pluginv2.StreamCapabilities, error) {
//	return s.server.GetStreamCapabilities(ctx, request)
//}

func (s streamGRPCServer) CanSubscribeToStream(ctx context.Context, request *pluginv2.SubscribeToStreamRequest) (*pluginv2.SubscribeToStreamResponse, error) {
	return s.server.CanSubscribeToStream(ctx, request)
}

func (s streamGRPCServer) RunStream(request *pluginv2.RunStreamRequest, server pluginv2.Stream_RunStreamServer) error {
	return s.server.RunStream(request, server)
}

//func (s streamGRPCServer) PublishToStream(ctx context.Context, request *pluginv2.StreamPublishRequest) (*pluginv2.StreamPublishResponse, error) {
//	return s.server.PublishToStream(ctx, request)
//}

type streamGRPCClient struct {
	client StreamClient
}

//func (s streamGRPCClient) InitStreams(ctx context.Context, in *pluginv2.StreamInitRequest, opts ...grpc.CallOption) (pluginv2.Stream_InitStreamsClient, error) {
//	return s.client.InitStreams(ctx, in, opts...)
//}
//
//func (s streamGRPCClient) GetStreamCapabilities(ctx context.Context, in *pluginv2.StreamCapabilitiesRequest, opts ...grpc.CallOption) (*pluginv2.StreamCapabilities, error) {
//	return s.client.GetStreamCapabilities(ctx, in, opts...)
//}

func (s streamGRPCClient) CanSubscribeToStream(ctx context.Context, in *pluginv2.SubscribeToStreamRequest, opts ...grpc.CallOption) (*pluginv2.SubscribeToStreamResponse, error) {
	return s.client.CanSubscribeToStream(ctx, in, opts...)
}

func (s streamGRPCClient) RunStream(ctx context.Context, in *pluginv2.RunStreamRequest, opts ...grpc.CallOption) (pluginv2.Stream_RunStreamClient, error) {
	return s.client.RunStream(ctx, in, opts...)
}

//func (s streamGRPCClient) PublishToStream(ctx context.Context, in *pluginv2.StreamPublishRequest, opts ...grpc.CallOption) (*pluginv2.StreamPublishResponse, error) {
//	return s.client.PublishToStream(ctx, in, opts...)
//}

var _ StreamServer = &streamGRPCServer{}
var _ StreamClient = &streamGRPCClient{}
