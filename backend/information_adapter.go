package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// informationSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type informationSDKAdapter struct {
	schemaHandler SchemaHandler
}

func newInformationSDKAdapter(schemaHandler SchemaHandler) *informationSDKAdapter {
	return &informationSDKAdapter{
		schemaHandler: schemaHandler,
	}
}

func (a *informationSDKAdapter) Schema(ctx context.Context, protoReq *pluginv2.SchemaRequest) (*pluginv2.SchemaResponse, error) {
	if a.schemaHandler != nil {
		parsedReq := FromProto().SchemaRequest(protoReq)
		resp, err := a.schemaHandler.Schema(ctx, parsedReq)
		if err != nil {
			return nil, err
		}

		return ToProto().SchemaResponse(resp), nil
	}

	return nil, nil
}
