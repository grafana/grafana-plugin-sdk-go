package backend

import (
	"context"
	"errors"
	"fmt"
)

// NewErrorSourceMiddleware returns a new backend.HandlerMiddleware that sets the error source in the
// context.Context, based on returned errors or query data response errors.
// If at least one query data response has a "downstream" error source and there isn't one with a "plugin" error source,
// the error source in the context is set to "downstream".
func NewErrorSourceMiddleware() HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &ErrorSourceMiddleware{
			BaseHandler: NewBaseHandler(next),
		}
	})
}

type ErrorSourceMiddleware struct {
	BaseHandler
}

func (m *ErrorSourceMiddleware) handleDownstreamError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	if IsDownstreamError(err) {
		if innerErr := WithDownstreamErrorSource(ctx); innerErr != nil {
			return fmt.Errorf("failed to set downstream error source: %w", errors.Join(innerErr, err))
		}
	}

	return err
}

func (m *ErrorSourceMiddleware) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	resp, err := m.BaseHandler.QueryData(ctx, req)
	err = m.handleDownstreamError(ctx, err)

	if err != nil {
		return resp, err
	} else if resp == nil || len(resp.Responses) == 0 {
		return nil, errors.New("both response and error are nil, but one must be provided")
	}

	// Set downstream error source in the context if there's at least one response with downstream error source,
	// and if there's no plugin error
	var hasPluginError bool
	var hasDownstreamError bool
	for _, r := range resp.Responses {
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
	}

	// A plugin error has higher priority than a downstream error,
	// so set to downstream only if there's no plugin error
	if hasDownstreamError && !hasPluginError {
		if err := WithDownstreamErrorSource(ctx); err != nil {
			return resp, fmt.Errorf("failed to set downstream status source: %w", err)
		}
	}

	return resp, err
}

func (m *ErrorSourceMiddleware) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	err := m.BaseHandler.CallResource(ctx, req, sender)
	return m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	resp, err := m.BaseHandler.CheckHealth(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) CollectMetrics(ctx context.Context, req *CollectMetricsRequest) (*CollectMetricsResult, error) {
	resp, err := m.BaseHandler.CollectMetrics(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) SubscribeStream(ctx context.Context, req *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	resp, err := m.BaseHandler.SubscribeStream(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) PublishStream(ctx context.Context, req *PublishStreamRequest) (*PublishStreamResponse, error) {
	resp, err := m.BaseHandler.PublishStream(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) RunStream(ctx context.Context, req *RunStreamRequest, sender *StreamSender) error {
	err := m.BaseHandler.RunStream(ctx, req, sender)
	return m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) ValidateAdmission(ctx context.Context, req *AdmissionRequest) (*ValidationResponse, error) {
	resp, err := m.BaseHandler.ValidateAdmission(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) MutateAdmission(ctx context.Context, req *AdmissionRequest) (*MutationResponse, error) {
	resp, err := m.BaseHandler.MutateAdmission(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}

func (m *ErrorSourceMiddleware) ConvertObjects(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	resp, err := m.BaseHandler.ConvertObjects(ctx, req)
	return resp, m.handleDownstreamError(ctx, err)
}
