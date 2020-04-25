package datasource

import (
	"context"

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
	m map[string]backend.QueryDataHandler
}

// NewQueryTypeMux allocates and returns a new QueryTypeMux.
func NewQueryTypeMux() *QueryTypeMux {
	return new(QueryTypeMux)
}

// Handle registers the handler for the given query type.
// If a handler already exists for query type, Handle panics.
func (mux *QueryTypeMux) Handle(queryType string, handler backend.QueryDataHandler) {
	if queryType == "" {
		panic("datasource: invalid query type")
	}
	if handler == nil {
		panic("datasource: nil handler")
	}
	if _, exist := mux.m[queryType]; exist {
		panic("datasource: multiple registrations for " + queryType)
	}

	if mux.m == nil {
		mux.m = map[string]backend.QueryDataHandler{}
	}

	mux.m[queryType] = handler
}

// HandleFunc registers the handler function for the given query type.
func (mux *QueryTypeMux) HandleFunc(queryType string, handler func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error)) {
	mux.Handle(queryType, QueryTypeHandlerFunc(handler))
}

// QueryData dispatches the request to the handler(s) whose
// query type matches the request queries query type.
func (mux *QueryTypeMux) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	responses := backend.Responses{}

	for _, q := range req.Queries {
		if handler, exists := mux.m[q.QueryType]; exists {
			qtResponse, err := handler.QueryData(ctx, req)
			if err != nil {
				return nil, err
			}
			for k, v := range qtResponse.Responses {
				responses[k] = v
			}
		}
	}

	return &backend.QueryDataResponse{
		Responses: responses,
	}, nil
}
