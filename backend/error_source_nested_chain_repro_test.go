package backend_test

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/stretchr/testify/require"
)

// nestedChainMiddleware reproduces the exact pattern used by
// grafana-enterprise's mtSettingMiddleware
// (pkg/extensions/datasource/pluginclient/mt_setting_middleware.go): at
// request time it builds its own inner middleware chain via
// backend.HandlerFromMiddlewares and calls .QueryData() on the resulting
// *MiddlewareHandler, instead of just calling next.QueryData() directly.
// This is a supported, public API usage -- backend.HandlerFromMiddlewares is
// exported precisely so callers can compose sub-chains -- so it must not
// silently break error-source propagation for whatever wraps it.
type nestedChainMiddleware struct {
	inner []backend.HandlerMiddleware
}

func (m *nestedChainMiddleware) CreateHandlerMiddleware(next backend.Handler) backend.Handler {
	return &nestedChainHandler{BaseHandler: backend.NewBaseHandler(next), next: next, inner: m.inner}
}

type nestedChainHandler struct {
	backend.BaseHandler
	next  backend.Handler
	inner []backend.HandlerMiddleware
}

func (h *nestedChainHandler) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	chain, err := backend.HandlerFromMiddlewares(h.next, h.inner...)
	if err != nil {
		return nil, err
	}
	return chain.QueryData(ctx, req)
}

// TestInitSource_SurvivesNestedMiddlewareHandler reproduces the real
// production bug end to end at SDK scale:
//
//	outer metrics middleware (reads context AFTER calling next)
//	  -> nestedChainMiddleware (builds its own inner chain via
//	     backend.HandlerFromMiddlewares and calls .QueryData() on it --
//	     exactly what mtSettingMiddleware.buildMiddlewareChain does)
//	    -> backend.NewErrorSourceMiddleware() (inside the inner chain)
//	      -> plugin handler (returns a single ref, ErrorSource=Downstream)
//
// Before the InitSource fix: the inner backend.HandlerFromMiddlewares call
// re-runs MiddlewareHandler.setupContext, which called the old,
// non-idempotent InitSource and attached a brand new *Source pointer. The
// inner ErrorSourceMiddleware then correctly mutates *that* pointer to
// Downstream, but the outer middleware is holding a reference to the
// *original* pointer from the outermost setupContext call, which nothing
// ever touches -- so it stays at its default, Plugin. This exactly
// reproduces the live production symptom: mt_app_grafana_datasources_cloud_requests_total
// and grafana_plugin_request_total (both recorded by middlewares outside
// grafana-enterprise's NewMTSettingMiddleware) show status_source=plugin for
// a request whose "Partial data response error" log line, from inside the
// middleware NewMTSettingMiddleware wraps, correctly says statusSource=downstream.
//
// After the fix, InitSource is idempotent: the inner setupContext call finds
// the outer context already has a *Source attached and reuses it, so the
// inner ErrorSourceMiddleware's mutation is the *same* pointer the outer
// middleware reads.
func TestInitSource_SurvivesNestedMiddlewareHandler(t *testing.T) {
	someErr := errors.New("oops")

	outer := &outerSourceCapturingMiddleware{}

	cdt := handlertest.NewHandlerMiddlewareTest(t,
		handlertest.WithMiddlewares(
			backend.HandlerMiddlewareFunc(func(next backend.Handler) backend.Handler {
				outer.BaseHandler = backend.NewBaseHandler(next)
				return outer
			}),
			&nestedChainMiddleware{
				inner: []backend.HandlerMiddleware{
					backend.NewErrorSourceMiddleware(),
				},
			},
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
		"outer middleware should observe the downstream source set by the inner ErrorSourceMiddleware, "+
			"even though it runs inside a nested backend.HandlerFromMiddlewares chain built at request time")
}
