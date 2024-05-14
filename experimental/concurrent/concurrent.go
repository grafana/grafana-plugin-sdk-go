package concurrent

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"golang.org/x/sync/errgroup"
)

type splitResponse struct {
	response backend.DataResponse
	refID    string
}

// SingleQuery is a single query to be executed concurrently
// Index is the index of the query in the request
// PluginContext is the plugin context
// Headers are the HTTP headers of the request
// DataQuery is the query to be executed
type SingleQuery struct {
	Index         int
	PluginContext backend.PluginContext
	Headers       http.Header
	DataQuery     backend.DataQuery
}

// SingleQueryData is the function that plugins need to define to execute a single query
type SingleQueryData func(ctx context.Context, query SingleQuery) (res backend.DataResponse)

// QueryData executes all queries from a request concurrently, using the provided function to execute each query.
// The concurrency limit is set by the limit parameter.
func QueryData(ctx context.Context, req *backend.QueryDataRequest, fn SingleQueryData, limit int) (*backend.QueryDataResponse, error) {
	headers := req.GetHTTPHeaders()
	ctxLogger := log.DefaultLogger.FromContext(ctx)
	ctxLogger.Debug("Concurrent QueryData", "queries", len(req.Queries))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	rchan := make(chan splitResponse, len(req.Queries))

	recoveryFn := func(q backend.DataQuery) {
		if r := recover(); r != nil {
			var err error
			ctxLogger.Error("query datasource panic", "error", r)
			if theErr, ok := r.(error); ok {
				err = theErr
			} else if theErrString, ok := r.(string); ok {
				err = fmt.Errorf(theErrString)
			} else {
				err = fmt.Errorf("unexpected error - %v", err)
			}
			// Due to the panic, there is no valid response for any query for this datasource. Append an error for each one.
			rchan <- splitResponse{backend.ErrDataResponse(backend.StatusInternal, err.Error()), q.RefID}
		}
	}

	// Execute each query and store the results by query RefID
	for i, q := range req.Queries {
		iIndex := i
		iQuery := q
		g.Go(func() error {
			// Handle panics from the query execution
			defer recoveryFn(iQuery)

			ctxLogger.Debug("Starting single query", "query", iQuery.RefID)
			res := fn(ctx, SingleQuery{
				Index:         iIndex,
				PluginContext: req.PluginContext,
				Headers:       headers,
				DataQuery:     iQuery,
			})
			ctxLogger.Debug("Finished single query", "query", iQuery.RefID)
			rchan <- splitResponse{res, iQuery.RefID}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(rchan)

	response := backend.NewQueryDataResponse()
	for result := range rchan {
		response.Responses[result.refID] = result.response
	}

	return response, nil
}
