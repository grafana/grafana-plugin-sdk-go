package datasource

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// The QueryTypeHandlerFunc type is an adapter to allow the use of
// ordinary functions as backend.QueryDataHandler. If f is a function
// with the appropriate signature, QueryTypeHandlerFunc(f) is a
// Handler that calls f.
type QueryTypeHandlerFunc func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error)

// QueryData calls f(ctx, req).
func (fn QueryTypeHandlerFunc) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return fn(ctx, req)
}

// QueryTypeMux is a query type multiplexer.
type QueryTypeMux struct {
	m               map[string]backend.QueryDataHandler
	fallbackHandler backend.QueryDataHandler
}

// NewQueryTypeMux allocates and returns a new QueryTypeMux.
func NewQueryTypeMux() *QueryTypeMux {
	return new(QueryTypeMux)
}

// Handle registers the handler for the given query type.
//
// Providing an empty queryType registers the handler as a fallback handler
// that will be called when query type doesn't match any registered handlers.
// If handler is nil, Handle panics.
// If a handler already exists for queryType, Handle panics.
func (mux *QueryTypeMux) Handle(queryType string, handler backend.QueryDataHandler) {
	if handler == nil {
		panic("datasource: nil handler")
	}

	if mux.m == nil {
		mux.m = map[string]backend.QueryDataHandler{}
		mux.fallbackHandler = QueryTypeHandlerFunc(fallbackHandler)
	}

	if _, exist := mux.m[queryType]; exist {
		panic("datasource: multiple registrations for " + queryType)
	}

	if queryType == "" {
		log.DefaultLogger.Debug("datasource: registering query type fallback handler")
		mux.fallbackHandler = handler
		return
	}

	log.DefaultLogger.Debug("datasource: registering query type handler", "queryType", queryType)
	mux.m[queryType] = handler
}

// HandleFunc registers the handler function for the given query type.
//
// Providing an empty queryType registers the handler as a fallback handler
// that will be called when query type doesn't match any registered handlers.
// If handler is nil, Handle panics.
// If a handler already exists for queryType, Handle panics.
func (mux *QueryTypeMux) HandleFunc(queryType string, handler func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error)) {
	mux.Handle(queryType, QueryTypeHandlerFunc(handler))
}

// QueryData dispatches the request to the handler(s) whose
// query type matches the request queries query type.
func (mux *QueryTypeMux) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	type requestHandler struct {
		handler backend.QueryDataHandler
		request *backend.QueryDataRequest
	}

	requests := map[string]requestHandler{}
	for _, q := range req.Queries {
		qt, handler := mux.getHandler(q.QueryType)
		if _, exists := requests[qt]; !exists {
			requests[qt] = requestHandler{
				handler: handler,
				request: &backend.QueryDataRequest{
					PluginContext: req.PluginContext,
					Headers:       req.Headers,
					Queries:       []backend.DataQuery{},
				},
			}
		}
		requests[qt].request.Queries = append(requests[qt].request.Queries, q)
	}

	responses := backend.Responses{}
	for _, rh := range requests {
		qtResponse, err := rh.handler.QueryData(ctx, rh.request)
		if err != nil {
			return nil, err
		}

		for k, v := range qtResponse.Responses {
			responses[k] = v
		}
	}

	return &backend.QueryDataResponse{
		Responses: responses,
	}, nil
}

func (mux *QueryTypeMux) getHandler(queryType string) (string, backend.QueryDataHandler) {
	handler, exists := mux.m[queryType]
	if !exists {
		return "", mux.fallbackHandler
	}

	return queryType, handler
}

func fallbackHandler(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	responses := backend.Responses{}
	for _, q := range req.Queries {
		responses[q.RefID] = backend.DataResponse{
			Error: fmt.Errorf("no handler found for query type '%s'", q.QueryType),
		}
	}

	return &backend.QueryDataResponse{
		Responses: responses,
	}, nil
}
