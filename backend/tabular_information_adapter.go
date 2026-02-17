package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// tabularInformationSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type tabularInformationSDKAdapter struct {
	tabularInformationHandler TabularInformationHandler
}

func newTabularInformationSDKAdapter(tabularInformationHandler TabularInformationHandler) *tabularInformationSDKAdapter {
	return &tabularInformationSDKAdapter{
		tabularInformationHandler: tabularInformationHandler,
	}
}

func (a *tabularInformationSDKAdapter) Tables(ctx context.Context, protoReq *pluginv2.TableInformationRequest) (*pluginv2.TableInformationResponse, error) {
	if a.tabularInformationHandler != nil {
		parsedReq := FromProto().TableInformationRequest(protoReq)
		resp, err := a.tabularInformationHandler.Tables(ctx, parsedReq)
		if err != nil {
			return nil, err
		}
		return ToProto().TableInformationResponse(resp), nil
	}
	return nil, nil
}
