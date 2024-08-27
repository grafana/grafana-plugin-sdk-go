package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler QueryDataHandler
}

func newDataSDKAdapter(handler QueryDataHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler: handler,
	}
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	ctx = setupContext(ctx, EndpointQueryData)
	parsedReq := FromProto().QueryDataRequest(req)

	var resp *QueryDataResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		ctx = withHeaderMiddleware(ctx, parsedReq.GetHTTPHeaders())
		var innerErr error
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
