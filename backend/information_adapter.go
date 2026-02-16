package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// informationSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type informationSDKAdapter struct {
	informationHandler InformationHandler
}

func newInformationSDKAdapter(informationHandler InformationHandler) *informationSDKAdapter {
	return &informationSDKAdapter{
		informationHandler: informationHandler,
	}
}

func (a *informationSDKAdapter) Tables(ctx context.Context, protoReq *pluginv2.TableInformationRequest) (*pluginv2.TableInformationResponse, error) {
	if a.informationHandler != nil {
		parsedReq := FromProto().TableInformationRequest(protoReq)
		resp, err := a.informationHandler.Tables(ctx, parsedReq)
		if err != nil {
			return nil, err
		}
		return ToProto().TableInformationResponse(resp), nil
	}
	return nil, nil
}
