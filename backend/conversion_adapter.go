package backend

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type conversionSDKAdapter struct {
	handler                ConversionHandler
	queryConversionHandler QueryConversionHandler
}

func newConversionSDKAdapter(handler ConversionHandler, queryConversionHandler QueryConversionHandler) *conversionSDKAdapter {
	return &conversionSDKAdapter{
		handler:                handler,
		queryConversionHandler: queryConversionHandler,
	}
}

func isQueryConversionRequest(req *ConversionRequest) bool {
	return req.TargetVersion.Group == "query.grafana.app" && req.TargetVersion.Version == "v0alpha1"
}

func (a *conversionSDKAdapter) ConvertQueryDataFromObjects(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	resp := &ConversionResponse{}
	queries := make([]any, 0, len(req.Objects))
	for _, obj := range req.Objects {
		if obj.ContentType != "application/json" {
			return nil, fmt.Errorf("unsupported content type %s", obj.ContentType)
		}
		input := &DataQuery{}
		err := json.Unmarshal(obj.Raw, input)
		if err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		res, err := a.queryConversionHandler.ConvertQuery(ctx, &QueryConversionRequest{
			PluginContext: req.PluginContext,
			Query:         *input,
		})
		if err != nil {
			return nil, err
		}
		queries = append(queries, res.Query)
	}

	for _, q := range queries {
		newJSON, err := json.Marshal(q)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		resp.Objects = append(resp.Objects, RawObject{
			Raw:         newJSON,
			ContentType: "application/json",
		})
	}
	return resp, nil
}

func (a *conversionSDKAdapter) ConvertObjects(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.ConversionResponse, error) {
	ctx = setupContext(ctx, EndpointConvertObject)
	parsedReq := FromProto().ConversionRequest(req)

	resp := &ConversionResponse{}
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		if isQueryConversionRequest(parsedReq) {
			if a.queryConversionHandler == nil {
				return RequestStatusError, fmt.Errorf("no query conversion handler defined")
			}
			resp, innerErr = a.ConvertQueryDataFromObjects(ctx, parsedReq)
			return RequestStatusFromError(innerErr), innerErr
		}
		resp, innerErr = a.handler.ConvertObjects(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().ConversionResponse(resp), nil
}
