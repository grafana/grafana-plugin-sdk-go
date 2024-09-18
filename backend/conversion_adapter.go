package backend

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type conversionSDKAdapter struct {
	handler ConversionHandler
	// queryConversionHandler can be nil if the adapter does not support query conversion.
	queryConversionHandler QueryConversionHandler
}

func newConversionSDKAdapter(handler ConversionHandler, queryConversionHandler QueryConversionHandler) *conversionSDKAdapter {
	return &conversionSDKAdapter{
		handler:                handler,
		queryConversionHandler: queryConversionHandler,
	}
}

func parseAsQueryRequest(req *ConversionRequest) ([]*QueryDataRequest, error) {
	var requests []*QueryDataRequest
	for _, obj := range req.Objects {
		if obj.ContentType != "application/json" {
			return nil, fmt.Errorf("unsupported content type %s", obj.ContentType)
		}
		input := &QueryDataRequest{}
		err := json.Unmarshal(obj.Raw, input)
		if err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		input.PluginContext = req.PluginContext
		requests = append(requests, input)
	}
	return requests, nil
}

func (a *conversionSDKAdapter) convertQueryDataRequest(ctx context.Context, requests []*QueryDataRequest) (*ConversionResponse, error) {
	resp := &ConversionResponse{}
	queries := []any{}
	for _, req := range requests {
		res, err := a.queryConversionHandler.ConvertQueryDataRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		// Queries are flattened into a single array
		queries = append(queries, res.Queries...)
	}

	for _, req := range queries {
		newJSON, err := json.Marshal(req)
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
	ctx = setupContext(ctx, EndpointConvertObjects)
	parsedReq := FromProto().ConversionRequest(req)

	resp := &ConversionResponse{}
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		if a.queryConversionHandler != nil {
			// Try to parse it as a query data request
			reqs, err := parseAsQueryRequest(parsedReq)
			if err == nil {
				resp, innerErr = a.convertQueryDataRequest(ctx, reqs)
				return RequestStatusFromError(innerErr), innerErr
			}
			// The object cannot be parsed as a query data request, so we will try to convert it as a generic object
		}
		resp, innerErr = a.handler.ConvertObjects(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().ConversionResponse(resp), nil
}
