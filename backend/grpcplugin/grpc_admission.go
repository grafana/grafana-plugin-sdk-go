package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// AdmissionServer represents a data server.
type AdmissionServer interface {
	pluginv2.AdmissionServer
}

// AdmissionClient represents a data client.
type AdmissionClient interface {
	pluginv2.AdmissionClient
}

// AdmissionGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type AdmissionGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	AdmissionServer AdmissionServer
}

// GRPCServer registers p as a data gRPC server.
func (p *AdmissionGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterAdmissionServer(s, &admissionGRPCServer{
		server: p.AdmissionServer,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *AdmissionGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &admissionGRPCClient{client: pluginv2.NewAdmissionClient(c)}, nil
}

type admissionGRPCServer struct {
	server AdmissionServer
}

func (s *admissionGRPCServer) ProcessInstanceSettings(ctx context.Context, req *pluginv2.ProcessInstanceSettingsRequest) (*pluginv2.ProcessInstanceSettingsResponse, error) {
	return s.server.ProcessInstanceSettings(ctx, req)
}

func (s *admissionGRPCServer) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	return s.server.ValidateAdmission(ctx, req)
}

func (s *admissionGRPCServer) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	return s.server.MutateAdmission(ctx, req)
}

type admissionGRPCClient struct {
	client pluginv2.AdmissionClient
}

func (c *admissionGRPCClient) ProcessInstanceSettings(ctx context.Context, req *pluginv2.ProcessInstanceSettingsRequest, opts ...grpc.CallOption) (*pluginv2.ProcessInstanceSettingsResponse, error) {
	return c.client.ProcessInstanceSettings(ctx, req, opts...)
}

func (c *admissionGRPCClient) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest, opts ...grpc.CallOption) (*pluginv2.AdmissionResponse, error) {
	return c.client.ValidateAdmission(ctx, req, opts...)
}

func (c *admissionGRPCClient) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest, opts ...grpc.CallOption) (*pluginv2.AdmissionResponse, error) {
	return c.client.MutateAdmission(ctx, req, opts...)
}

var _ AdmissionServer = &admissionGRPCServer{}
var _ AdmissionClient = &admissionGRPCClient{}
