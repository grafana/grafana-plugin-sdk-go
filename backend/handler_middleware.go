package backend

import (
	"context"
	"errors"
)

var (
	errNilRequest = errors.New("req cannot be nil")
	errNilSender  = errors.New("sender cannot be nil")
)

// HandlerMiddleware is an interface representing the ability to create a middleware
// that implements the Handler interface.
type HandlerMiddleware interface {
	// CreateHandlerMiddleware creates a new Handler by decorating next Handler.
	CreateHandlerMiddleware(next Handler) Handler
}

// The HandlerMiddlewareFunc type is an adapter to allow the use of ordinary
// functions as HandlerMiddleware's. If f is a function with the appropriate
// signature, HandlerMiddlewareFunc(f) is a HandlerMiddleware that calls f.
type HandlerMiddlewareFunc func(next Handler) Handler

// CreateHandlerMiddleware implements the HandlerMiddleware interface.
func (fn HandlerMiddlewareFunc) CreateHandlerMiddleware(next Handler) Handler {
	return fn(next)
}

// MiddlewareHandler decorates a Handler with HandlerMiddleware's.
type MiddlewareHandler struct {
	handler Handler
}

// HandlerFromMiddlewares creates a new MiddlewareHandler implementing Handler that decorates finalHandler with middlewares.
func HandlerFromMiddlewares(finalHandler Handler, middlewares ...HandlerMiddleware) (*MiddlewareHandler, error) {
	if finalHandler == nil {
		return nil, errors.New("finalHandler cannot be nil")
	}

	return &MiddlewareHandler{
		handler: handlerFromMiddlewares(middlewares, finalHandler),
	}, nil
}

func (h *MiddlewareHandler) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.QueryData(ctx, req)
}

func (h MiddlewareHandler) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	if req == nil {
		return errNilRequest
	}

	if sender == nil {
		return errNilSender
	}

	return h.handler.CallResource(ctx, req, sender)
}

func (h MiddlewareHandler) CollectMetrics(ctx context.Context, req *CollectMetricsRequest) (*CollectMetricsResult, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.CollectMetrics(ctx, req)
}

func (h MiddlewareHandler) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.CheckHealth(ctx, req)
}

func (h MiddlewareHandler) SubscribeStream(ctx context.Context, req *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.SubscribeStream(ctx, req)
}

func (h MiddlewareHandler) PublishStream(ctx context.Context, req *PublishStreamRequest) (*PublishStreamResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.PublishStream(ctx, req)
}

func (h MiddlewareHandler) RunStream(ctx context.Context, req *RunStreamRequest, sender *StreamSender) error {
	if req == nil {
		return errNilRequest
	}

	if sender == nil {
		return errors.New("sender cannot be nil")
	}

	return h.handler.RunStream(ctx, req, sender)
}

func (h MiddlewareHandler) ValidateAdmission(ctx context.Context, req *AdmissionRequest) (*ValidationResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.ValidateAdmission(ctx, req)
}

func (h MiddlewareHandler) MutateAdmission(ctx context.Context, req *AdmissionRequest) (*MutationResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.MutateAdmission(ctx, req)
}

func (h MiddlewareHandler) ConvertObjects(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	if req == nil {
		return nil, errNilRequest
	}

	return h.handler.ConvertObjects(ctx, req)
}

func handlerFromMiddlewares(middlewares []HandlerMiddleware, finalHandler Handler) Handler {
	if len(middlewares) == 0 {
		return finalHandler
	}

	reversed := reverseMiddlewares(middlewares)
	next := finalHandler

	for _, m := range reversed {
		next = m.CreateHandlerMiddleware(next)
	}

	return next
}

func reverseMiddlewares(middlewares []HandlerMiddleware) []HandlerMiddleware {
	reversed := make([]HandlerMiddleware, len(middlewares))
	copy(reversed, middlewares)

	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}

	return reversed
}
