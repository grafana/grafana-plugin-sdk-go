package backend

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type conversionSDKAdapter struct {
	handler                ConversionHandler
	queryConversionHandler QueryConversionHandler // Optional
}

func newConversionSDKAdapter(handler ConversionHandler, queryConversionHandler QueryConversionHandler) *conversionSDKAdapter {
	return &conversionSDKAdapter{
		handler:                handler,
		queryConversionHandler: queryConversionHandler,
	}
}

func (a *conversionSDKAdapter) ConvertQueryDataFromObjects(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	if req.TargetVersion.Group != "query" || req.TargetVersion.Version != "v0alpha1" {
		return nil, fmt.Errorf("unsupported target version %s/%s", req.TargetVersion.Group, req.TargetVersion.Version)
	}

	resp := &ConversionResponse{}
	queries := make([]DataQuery, 0, len(req.Objects))
	for _, obj := range req.Objects {
		if obj.ContentType != "application/json" {
			return nil, fmt.Errorf("unsupported content type %s", obj.ContentType)
		}
		input := &DataQuery{}
		err := json.Unmarshal(obj.Raw, input)
		if err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		queries = append(queries, *input)
	}
	queryConversionRes, innerErr := a.queryConversionHandler.ConvertQuery(ctx, &QueryConversionRequest{
		Queries: queries,
	})
	if innerErr != nil {
		return nil, innerErr
	}
	for _, q := range queryConversionRes.Queries {
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
		if a.handler != nil {
			resp, innerErr = a.handler.ConvertObjects(ctx, parsedReq)
			return RequestStatusFromError(innerErr), innerErr
		}
		if a.queryConversionHandler != nil && GrafanaConfigFromContext(ctx).FeatureToggles().IsEnabled("dsQueryConvert") {
			resp, innerErr = a.ConvertQueryDataFromObjects(ctx, parsedReq)
			return RequestStatusFromError(innerErr), innerErr
		}
		return RequestStatusError, fmt.Errorf("no handler defined")
	})
	if err != nil {
		return nil, err
	}

	return ToProto().ConversionResponse(resp), nil
}
