package backend

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// metadataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type metadataSDKAdapter struct {
	provideMetadataHandler ProvideMetadataHandler
}

func newMetadataSDKAdapter(handler ProvideMetadataHandler) *metadataSDKAdapter {
	return &metadataSDKAdapter{
		provideMetadataHandler: handler,
	}
}

func (a *metadataSDKAdapter) ProvideMetadata(ctx context.Context, protoReq *pluginv2.ProvideMetadataRequest) (*pluginv2.ProvideMetadataResponse, error) {
	if a.provideMetadataHandler == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}

	ctx = propagateTenantIDIfPresent(ctx)
	resp, err := a.provideMetadataHandler.ProvideMetadata(ctx, FromProto().ProvideMetadataRequest(protoReq))
	if err != nil {
		return nil, err
	}
	return ToProto().ProvideMetadataResponse(resp), nil
}
