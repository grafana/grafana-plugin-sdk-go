package backend_test

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/stretchr/testify/require"
)

// outerSourceCapturingMiddleware mimics the shape of the two real, production
// metrics middlewares that sit *outside* backend.NewErrorSourceMiddleware()
// in the tempo-datasource-grafana-app request chain (Grafana OSS's
// clientmiddleware.MetricsMiddleware, which backs grafana_plugin_request_total,
// and grafana-enterprise's requestMetricsMiddleware, which backs
// mt_app_grafana_datasources_cloud_requests_total): call the inner handler
// first, then read backend.ErrorSourceFromContext(ctx) afterwards to decide
// what to record.
type outerSourceCapturingMiddleware struct {
	backend.BaseHandler
	capturedSource backend.ErrorSource
}

func (m *outerSourceCapturingMiddleware) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	resp, err := m.BaseHandler.QueryData(ctx, req)
	m.capturedSource = backend.ErrorSourceFromContext(ctx)
	return resp, err
}

// TestErrorSourceMiddleware_ObservedByOuterMiddleware reproduces, at minimal
// scale, the real production topology under investigation:
//
//	outer metrics middleware (reads context AFTER calling next)
//	  -> backend.NewErrorSourceMiddleware() (sets context from per-response ErrorSource)
//	    -> plugin handler (returns a single ref with ErrorSource=Downstream)
//
// This is exactly the scenario seen live in production for a real, currently
// failing request: a single QueryData ref (refID=A) whose DataResponse is
// correctly tagged ErrorSource=Downstream by the tempo datasource plugin
// (confirmed both by grafana-plugin-sdk-go/backend's own
// TestErrorSourceMiddleware/QueryData_response_with_errors/single_downstream_error
// case, and by the live "Partial data response error ... statusSource=downstream"
// log line), yet the aggregate metric recorded one layer out
// (mt_app_grafana_datasources_cloud_requests_total, fed by a middleware in
// this exact outer position) shows status_source=plugin in production.
//
// If this test passes, backend.NewErrorSourceMiddleware()'s context mutation
// is correctly visible to a middleware sitting outside it in a minimal
// two-middleware chain, which would mean the production bug is NOT in
// grafana-plugin-sdk-go's core middleware or in this basic topology, and must
// come from something specific to the full ~25-middleware chain assembled in
// grafana-enterprise's pkg/extensions/datasource/pluginclient/plugin_client.go
// (e.g. a middleware in between that derives a new context instead of
// threading the original one through, breaking WithSource's ability to find
// the *Source value backend.MiddlewareHandler.setupContext attached).
func TestErrorSourceMiddleware_ObservedByOuterMiddleware(t *testing.T) {
	someErr := errors.New("oops")

	outer := &outerSourceCapturingMiddleware{}

	cdt := handlertest.NewHandlerMiddlewareTest(t,
		handlertest.WithMiddlewares(
			backend.HandlerMiddlewareFunc(func(next backend.Handler) backend.Handler {
				outer.BaseHandler = backend.NewBaseHandler(next)
				return outer
			}),
			backend.NewErrorSourceMiddleware(),
		),
	)

	cdt.TestHandler.QueryDataFunc = func(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		return &backend.QueryDataResponse{
			Responses: map[string]backend.DataResponse{
				"A": {Error: someErr, ErrorSource: backend.ErrorSourceDownstream},
			},
		}, nil
	}

	_, err := cdt.MiddlewareHandler.QueryData(context.Background(), &backend.QueryDataRequest{})
	require.NoError(t, err, "QueryData itself should not fail just because one ref errored downstream")

	require.Equal(t, backend.ErrorSourceDownstream, outer.capturedSource,
		"outer middleware should observe the downstream source set by ErrorSourceMiddleware for a single, correctly-tagged downstream ref")
}
