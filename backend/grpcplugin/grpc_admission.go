package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// AdmissionServer represents a data server.
type AdmissionServer interface {
	pluginv2.AdmissionControlServer
}

// AdmissionClient represents a data client.
type AdmissionClient interface {
	pluginv2.AdmissionControlClient
}

// StorageGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type StorageGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	AdmissionServer AdmissionServer
}

// GRPCServer registers p as a data gRPC server.
func (p *StorageGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterAdmissionControlServer(s, &storageGRPCServer{
		server: p.AdmissionServer,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *StorageGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &storageGRPCClient{client: pluginv2.NewAdmissionControlClient(c)}, nil
}

type storageGRPCServer struct {
	server AdmissionServer
}

func (s *storageGRPCServer) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	return s.server.ValidateAdmission(ctx, req)
}

func (s *storageGRPCServer) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	return s.server.MutateAdmission(ctx, req)
}

func (s *storageGRPCServer) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.AdmissionResponse, error) {
	return s.server.ConvertObject(ctx, req)
}

type storageGRPCClient struct {
	client pluginv2.AdmissionControlClient
}

func (s *storageGRPCClient) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest, opts ...grpc.CallOption) (*pluginv2.AdmissionResponse, error) {
	return s.client.ValidateAdmission(ctx, req, opts...)
}

func (s *storageGRPCClient) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest, opts ...grpc.CallOption) (*pluginv2.AdmissionResponse, error) {
	return s.client.MutateAdmission(ctx, req, opts...)
}

func (s *storageGRPCClient) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest, opts ...grpc.CallOption) (*pluginv2.AdmissionResponse, error) {
	return s.client.ConvertObject(ctx, req, opts...)
}

var _ AdmissionServer = &storageGRPCServer{}
var _ AdmissionClient = &storageGRPCClient{}
