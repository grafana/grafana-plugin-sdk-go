package datasource

import (
	"context"
	"fmt"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// QueryTypeMux is a query type multiplexer.
type QueryTypeMux struct {
	m               map[string]backend.QueryDataHandler
	c               map[string]backend.QueryChunkedDataHandlerFunc
	fallbackHandler backend.QueryDataHandler
}

// NewQueryTypeMux allocates and returns a new QueryTypeMux.
func NewQueryTypeMux() *QueryTypeMux {
	return &QueryTypeMux{
		m:               make(map[string]backend.QueryDataHandler),
		c:               make(map[string]backend.QueryChunkedDataHandlerFunc),
		fallbackHandler: backend.QueryDataHandlerFunc(fallbackHandler),
	}
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
		mux.fallbackHandler = backend.QueryDataHandlerFunc(fallbackHandler)
	}

	if _, exist := mux.m[queryType]; exist {
		panic("datasource: multiple registrations for " + queryType)
	}

	if queryType == "" {
		mux.fallbackHandler = handler
		return
	}

	mux.m[queryType] = handler
}

// HandleFunc registers the handler function for the given query type.
//
// Providing an empty queryType registers the handler as a fallback handler
// that will be called when query type doesn't match any registered handlers.
// If handler is nil, Handle panics.
// If a handler already exists for queryType, Handle panics.
func (mux *QueryTypeMux) HandleFunc(queryType string, handler func(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error)) {
	mux.Handle(queryType, backend.QueryDataHandlerFunc(handler))
}

// HandleChunkedQueryType registers a chunked chandler function for a given query type.
func (mux *QueryTypeMux) HandleChunkedQueryType(queryType string, handler backend.QueryChunkedDataHandlerFunc) {
	if queryType == "" {
		panic("ChunkedHandleFunc: requires query type")
	}
	if handler == nil {
		panic("ChunkedHandleFunc: nil handler")
	}
	if _, exist := mux.c[queryType]; exist {
		panic("ChunkedHandleFunc: multiple registrations for " + queryType)
	}
	mux.c[queryType] = handler
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

// Process chunked data query
func (mux *QueryTypeMux) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) (err error) {
	var wg sync.WaitGroup
	for _, q := range req.Queries {
		wg.Go(func() {
			// When the query type has an explicit chunker, use that
			chunker, ok := mux.c[q.QueryType]
			if ok {
				if e := chunker(ctx, &backend.QueryChunkedDataRequest{
					PluginContext: req.PluginContext,
					Headers:       req.Headers,
					Queries:       []backend.DataQuery{q},
				}, w); e != nil {
					err = e // error returned from parent function
				}
				return
			}

			// Fallback to individual query handlers
			_, handler := mux.getHandler(q.QueryType)
			rsp, e := handler.QueryData(ctx, &backend.QueryDataRequest{
				PluginContext: req.PluginContext,
				Headers:       req.Headers,
				Queries:       []backend.DataQuery{q},
			})
			if e != nil {
				err = e // error returned from parent function
				return
			}

			for k, v := range rsp.Responses {
				for i, f := range v.Frames {
					f.RefID = k
					if e := w.WriteFrame(ctx, k, fmt.Sprintf("f%d", i), f); err != nil {
						err = e // error returned from parent function
					}
				}

				if v.Error != nil || (v.Status.IsValid() && v.Status != backend.StatusOK) {
					if e := w.WriteError(ctx, k, v.Status, v.Error); e != nil {
						err = e // error returned from parent function
					}
				}
			}
		})
	}
	wg.Wait()
	return err
}

func (mux *QueryTypeMux) getHandler(queryType string) (string, backend.QueryDataHandler) {
	handler, exists := mux.m[queryType]
	if !exists {
		return "", mux.fallbackHandler
	}

	return queryType, handler
}

func fallbackHandler(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	responses := backend.Responses{}
	for _, q := range req.Queries {
		responses[q.RefID] = backend.DataResponse{
			Error:       fmt.Errorf("no handler found for query type '%s'", q.QueryType),
			ErrorSource: backend.ErrorSourcePlugin,
		}
	}

	return &backend.QueryDataResponse{
		Responses: responses,
	}, nil
}
