package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// registrationSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type registrationSDKAdapter struct {
	registrationHandler RegistrationHandler
}

func newRegistrationSDKAdapter(handler RegistrationHandler) *registrationSDKAdapter {
	return &registrationSDKAdapter{
		registrationHandler: handler,
	}
}

// TODO test
func (r *registrationSDKAdapter) QueryRoles(ctx context.Context, req *pluginv2.QueryRolesRequest) (*pluginv2.QueryRolesResponse, error) {
	resp := r.registrationHandler.QueryRoles(ctx, FromProto().QueryRolesRequest(req))

	return ToProto().QueryRolesResponse(resp), nil
}
