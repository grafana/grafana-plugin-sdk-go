package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"google.golang.org/grpc"
)

type AccessControlClientMock struct{}

func (a *AccessControlClientMock) HasAccess(ctx context.Context, req *pluginv2.HasAccessRequest, opts ...grpc.CallOption) (*pluginv2.HasAccessResponse, error) {
	return &pluginv2.HasAccessResponse{HasAccess: true}, nil
}
