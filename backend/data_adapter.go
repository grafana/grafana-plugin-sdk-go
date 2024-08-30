package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler  QueryDataHandler
	conversionHandler ConversionHandler // Optional
}

func newDataSDKAdapter(handler QueryDataHandler, conversionHandler ConversionHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler:  handler,
		conversionHandler: conversionHandler,
	}
}

func (a *dataSDKAdapter) ConvertQueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataRequest, error) {
	convertRequest := &ConversionRequest{
		PluginContext: req.PluginContext,
		// TODO: What should we use for UID and TargetVersion?
		UID: uuid.New().String(),
		TargetVersion: GroupVersion{
			Group:   "query",
			Version: "v0alpha1",
		},
		Objects: make([]RawObject, 0, len(req.Queries)),
	}
	for _, q := range req.Queries {
		raw, err := json.Marshal(q)
		if err != nil {
			return nil, err
		}
		convertRequest.Objects = append(convertRequest.Objects, RawObject{
			Raw:         raw,
			ContentType: "application/json",
		})
	}
	convertResponse, err := a.conversionHandler.ConvertObjects(ctx, convertRequest)
	if err != nil {
		// TODO: Use convertResponse.Result to return an error?
		return nil, err
	}
	convertedQueries := make([]DataQuery, 0, len(convertResponse.Objects))
	for _, obj := range convertResponse.Objects {
		var q DataQuery
		if err := json.Unmarshal(obj.Raw, &q); err != nil {
			return nil, err
		}
		convertedQueries = append(convertedQueries, q)
	}
	req.Queries = convertedQueries

	return req, nil
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	ctx = setupContext(ctx, EndpointQueryData)
	parsedReq := FromProto().QueryDataRequest(req)

	var resp *QueryDataResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		ctx = withHeaderMiddleware(ctx, parsedReq.GetHTTPHeaders())
		var innerErr error
		if a.conversionHandler != nil {
			parsedReq, innerErr = a.ConvertQueryData(ctx, parsedReq)
			if innerErr != nil {
				return RequestStatusError, innerErr
			}
		}
		resp, innerErr = a.queryDataHandler.QueryData(ctx, parsedReq)

		if resp == nil || len(resp.Responses) == 0 {
			return RequestStatusFromError(innerErr), innerErr
		}

		if isCancelledError(innerErr) {
			return RequestStatusCancelled, nil
		}

		if isHTTPTimeoutError(innerErr) {
			return RequestStatusError, nil
		}

		// Set downstream status source in the context if there's at least one response with downstream status source,
		// and if there's no plugin error
		var hasPluginError bool
		var hasDownstreamError bool
		var hasCancelledError bool
		var hasHTTPTimeoutError bool
		for _, r := range resp.Responses {
			if r.Error == nil {
				continue
			}

			if isCancelledError(r.Error) {
				hasCancelledError = true
			}
			if isHTTPTimeoutError(r.Error) {
				hasHTTPTimeoutError = true
			}

			if r.ErrorSource == ErrorSourceDownstream {
				hasDownstreamError = true
			} else {
				hasPluginError = true
			}
		}

		if hasCancelledError {
			if err := WithDownstreamErrorSource(ctx); err != nil {
				return RequestStatusError, fmt.Errorf("failed to set downstream status source: %w", errors.Join(innerErr, err))
			}
			return RequestStatusCancelled, nil
		}

		if hasHTTPTimeoutError {
			if err := WithDownstreamErrorSource(ctx); err != nil {
				return RequestStatusError, fmt.Errorf("failed to set downstream status source: %w", errors.Join(innerErr, err))
			}
			return RequestStatusError, nil
		}

		// A plugin error has higher priority than a downstream error,
		// so set to downstream only if there's no plugin error
		if hasDownstreamError && !hasPluginError {
			if err := WithDownstreamErrorSource(ctx); err != nil {
				return RequestStatusError, fmt.Errorf("failed to set downstream status source: %w", errors.Join(innerErr, err))
			}
			return RequestStatusError, nil
		}

		if hasPluginError {
			if err := WithErrorSource(ctx, ErrorSourcePlugin); err != nil {
				return RequestStatusError, fmt.Errorf("failed to set plugin status source: %w", errors.Join(innerErr, err))
			}
			return RequestStatusError, nil
		}

		if innerErr != nil {
			return RequestStatusFromError(innerErr), innerErr
		}

		return RequestStatusOK, nil
	})
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
