package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
	"github.com/stretchr/testify/require"
)

func TestErrorSourceMiddleware(t *testing.T) {
	t.Run("With non-downstream HTTP error returned from http.RoundTripper should not be wrapped in a downstream error", func(t *testing.T) {
		ctx := &testContext{}
		someErr := errors.New("some error")
		finalRoundTripper := ctx.createRoundTripperWithError(someErr)
		mw := ErrorSourceMiddleware()
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, ErrorSourceMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.Error(t, err)
		if resp.Body != nil {
			require.NoError(t, resp.Body.Close())
		}
		require.False(t, status.IsDownstreamError(err))
		require.ErrorIs(t, err, someErr)
	})

	t.Run("With downstream HTTP error returned from http.RoundTripper should be wrapped in a downstream error", func(t *testing.T) {
		ctx := &testContext{}
		someErr := &net.DNSError{IsNotFound: true}
		finalRoundTripper := ctx.createRoundTripperWithError(someErr)
		mw := ErrorSourceMiddleware()
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, ErrorSourceMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.Error(t, err)
		if resp.Body != nil {
			require.NoError(t, resp.Body.Close())
		}
		require.True(t, status.IsDownstreamError(err))
		require.ErrorIs(t, err, someErr)
	})

	t.Run("With downstream HTTP error the request context source is set to downstream", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripperWithError(&net.DNSError{IsNotFound: true})
		rt := ErrorSourceMiddleware().CreateMiddleware(Options{}, finalRoundTripper)

		reqCtx := status.InitSource(context.Background())
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "http://", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.Error(t, err)
		if resp.Body != nil {
			require.NoError(t, resp.Body.Close())
		}
		require.Equal(t, status.SourceDownstream, status.SourceFromContext(reqCtx))
	})

	t.Run("With non-downstream HTTP error the request context source is not changed", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripperWithError(errors.New("some error"))
		rt := ErrorSourceMiddleware().CreateMiddleware(Options{}, finalRoundTripper)

		reqCtx := status.InitSource(context.Background())
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "http://", nil)
		require.NoError(t, err)
		resp, err := rt.RoundTrip(req)
		require.Error(t, err)
		if resp.Body != nil {
			require.NoError(t, resp.Body.Close())
		}
		require.Equal(t, status.SourcePlugin, status.SourceFromContext(reqCtx))
	})
}
