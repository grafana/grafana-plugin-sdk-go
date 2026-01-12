package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// querySchemaSDKAdapter adapts the SDK QuerySchemaHandler to the gRPC interface.
type querySchemaSDKAdapter struct {
	handler QuerySchemaHandler
}

func newQuerySchemaSDKAdapter(handler QuerySchemaHandler) *querySchemaSDKAdapter {
	return &querySchemaSDKAdapter{
		handler: handler,
	}
}

func (a *querySchemaSDKAdapter) GetQuerySchema(ctx context.Context, req *pluginv2.GetQuerySchemaRequest) (*pluginv2.GetQuerySchemaResponse, error) {
	parsedReq := FromProto().GetQuerySchemaRequest(req)
	resp, err := a.handler.GetQuerySchema(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().GetQuerySchemaResponse(resp), nil
}
