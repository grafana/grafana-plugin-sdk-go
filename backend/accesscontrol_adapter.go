package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// accesscontrolSDKAdpater adapter between low level plugin protocol and SDK interfaces.
type accesscontrolSDKAdpater struct {
	accessControlHandler AccessControl
}

func NewAccesscontrolSDKAdpater(handler AccessControl) *accesscontrolSDKAdpater {
	return &accesscontrolSDKAdpater{
		accessControlHandler: handler,
	}
}

func (a *accesscontrolSDKAdpater) HasAccess(ctx context.Context, req *pluginv2.HasAccessRequest) (*pluginv2.HasAccessResponse, error) {
	// Convert req to SDK req
	sdkReq := FromProto().HasAccessRequest(req)

	resp, err := a.accessControlHandler.HasAccess(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	return ToProto().HasAccessResponse(resp), nil
}
