package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// accesscontrolSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type accesscontrolSDKAdapter struct {
	hasAccessHandler HasAccessHandler
}

func newAccesscontrolSDKAdapter(handler HasAccessHandler) *accesscontrolSDKAdapter {
	return &accesscontrolSDKAdapter{
		hasAccessHandler: handler,
	}
}

func (a *accesscontrolSDKAdapter) HasAccess(ctx context.Context, req *pluginv2.HasAccessRequest) (*pluginv2.HasAccessResponse, error) {
	resp, err := a.hasAccessHandler.HasAccess(ctx, FromProto().HasAccessRequest(req))
	if err != nil {
		return nil, err
	}

	return ToProto().HasAccessResponse(resp), nil
}
