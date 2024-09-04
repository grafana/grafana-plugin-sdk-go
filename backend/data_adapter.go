package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler       QueryDataHandler
	queryConversionHandler QueryConversionHandler // Optional
}

func newDataSDKAdapter(handler QueryDataHandler, queryConversionHandler QueryConversionHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler:       handler,
		queryConversionHandler: queryConversionHandler,
	}
}

func (a *dataSDKAdapter) ConvertQueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataRequest, error) {
	convertRequest := &QueryConversionRequest{
		PluginContext: req.PluginContext,
		Queries:       req.Queries,
	}
	convertResponse, err := a.queryConversionHandler.ConvertQuery(ctx, convertRequest)
	if err != nil {
		return nil, err
	}
	req.Queries = convertResponse.Queries

	return req, nil
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	ctx = setupContext(ctx, EndpointQueryData)
	parsedReq := FromProto().QueryDataRequest(req)

	var resp *QueryDataResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		ctx = withHeaderMiddleware(ctx, parsedReq.GetHTTPHeaders())
		var innerErr error
		if a.queryConversionHandler != nil && GrafanaConfigFromContext(ctx).FeatureToggles().IsEnabled("dsQueryConvert") {
			parsedReq, innerErr = a.ConvertQueryData(ctx, parsedReq)
			if innerErr != nil {
				return RequestStatusError, innerErr
			}
		}
		resp, innerErr = a.queryDataHandler.QueryData(ctx, parsedReq)

		status := RequestStatusFromQueryDataResponse(resp, innerErr)
		if innerErr != nil {
			return status, innerErr
		} else if resp == nil {
			return RequestStatusError, errors.New("both response and error are nil, but one must be provided")
		}
		ctxLogger := Logger.FromContext(ctx)

		// Set downstream status source in the context if there's at least one response with downstream status source,
		// and if there's no plugin error
		var hasPluginError, hasDownstreamError bool
		for refID, r := range resp.Responses {
			if r.Error == nil {
				continue
			}

			// if error source not set and the error is a downstream error, set error source to downstream.
			if !r.ErrorSource.IsValid() && IsDownstreamError(r.Error) {
				r.ErrorSource = ErrorSourceDownstream
			}

			if !r.Status.IsValid() {
				r.Status = statusFromError(r.Error)
			}

			if r.ErrorSource == ErrorSourceDownstream {
				hasDownstreamError = true
			} else {
				hasPluginError = true
			}

			logParams := []any{
				"refID", refID,
				"status", int(r.Status),
				"error", r.Error,
				"statusSource", string(r.ErrorSource),
			}
			ctxLogger.Error("Partial data response error", logParams...)
		}

		// A plugin error has higher priority than a downstream error,
		// so set to downstream only if there's no plugin error
		if hasPluginError {
			if err := WithErrorSource(ctx, ErrorSourcePlugin); err != nil {
				return RequestStatusError, fmt.Errorf("failed to set plugin status source: %w", errors.Join(innerErr, err))
			}
		} else if hasDownstreamError {
			if err := WithDownstreamErrorSource(ctx); err != nil {
				return RequestStatusError, fmt.Errorf("failed to set downstream status source: %w", errors.Join(innerErr, err))
			}
		}

		return status, nil
	})
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
