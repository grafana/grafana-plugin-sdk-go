package httpclient

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("New() without any opts should return expected http client and middlewares", func(t *testing.T) {
		client, err := New()
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NotSame(t, http.DefaultClient, client)

		require.Equal(t, 30*time.Second, client.Timeout)
		require.NotSame(t, &http.DefaultTransport, &client.Transport)
	})

	t.Run("New() with opts and no middlewares specified should return expected http client and middlewares", func(t *testing.T) {
		client, err := New(Options{})
		require.NoError(t, err)
		require.NotNil(t, client)
		require.NotSame(t, http.DefaultClient, client)

		require.Equal(t, 30*time.Second, client.Timeout)
		require.NotSame(t, &http.DefaultTransport, &client.Transport)
	})

	t.Run("New() with opts and empty middlewares should return expected http client and transport", func(t *testing.T) {
		client, err := New(Options{
			Timeouts: &TimeoutOptions{
				Timeout:               time.Second,
				DialTimeout:           7 * time.Second,
				KeepAlive:             2 * time.Second,
				TLSHandshakeTimeout:   3 * time.Second,
				ExpectContinueTimeout: 4 * time.Second,
				MaxConnsPerHost:       10,
				MaxIdleConns:          5,
				MaxIdleConnsPerHost:   7,
				IdleConnTimeout:       6 * time.Second,
			},
			Middlewares: []Middleware{},
		})
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, time.Second, client.Timeout)

		// this only works when there are no middlewares, otherwise the transport is wrapped
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport)
		require.Equal(t, 3*time.Second, transport.TLSHandshakeTimeout)
		require.Equal(t, 4*time.Second, transport.ExpectContinueTimeout)
		require.Equal(t, 10, transport.MaxConnsPerHost)
		require.Equal(t, 5, transport.MaxIdleConns)
		require.Equal(t, 7, transport.MaxIdleConnsPerHost)
		require.Equal(t, 6*time.Second, transport.IdleConnTimeout)
	})

	t.Run("New() with non-empty opts should use default middleware", func(t *testing.T) {
		usedMiddlewares := []Middleware{}
		client, err := New(Options{ConfigureMiddleware: func(_ Options, existingMiddleware []Middleware) []Middleware {
			usedMiddlewares = existingMiddleware
			return existingMiddleware
		}})
		require.NoError(t, err)
		require.NotNil(t, client)

		require.Len(t, usedMiddlewares, 6)
		require.Equal(t, TracingMiddlewareName, usedMiddlewares[0].(MiddlewareName).MiddlewareName())
		require.Equal(t, DataSourceMetricsMiddlewareName, usedMiddlewares[1].(MiddlewareName).MiddlewareName())
		require.Equal(t, BasicAuthenticationMiddlewareName, usedMiddlewares[2].(MiddlewareName).MiddlewareName())
		require.Equal(t, CustomHeadersMiddlewareName, usedMiddlewares[3].(MiddlewareName).MiddlewareName())
		require.Equal(t, ContextualMiddlewareName, usedMiddlewares[4].(MiddlewareName).MiddlewareName())
		require.Equal(t, ErrorSourceMiddlewareName, usedMiddlewares[5].(MiddlewareName).MiddlewareName())
	})

	t.Run("New() with opts middleware should return expected http.Client", func(t *testing.T) {
		ctx := &testContext{}
		usedMiddlewares := []Middleware{}
		client, err := New(Options{
			Middlewares: []Middleware{ctx.createMiddleware("mw1"), ctx.createMiddleware("mw2"), ctx.createMiddleware("mw3")},
			ConfigureMiddleware: func(_ Options, existingMiddleware []Middleware) []Middleware {
				middlewares := existingMiddleware
				for i, j := 0, len(existingMiddleware)-1; i < j; i, j = i+1, j-1 {
					middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
				}
				usedMiddlewares = middlewares
				return middlewares
			},
		})
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, DefaultTimeoutOptions.Timeout, client.Timeout)

		t.Run("Should use configured middlewares and implement MiddlewareName", func(t *testing.T) {
			require.Len(t, usedMiddlewares, 3)
			require.Equal(t, "mw3", usedMiddlewares[0].(MiddlewareName).MiddlewareName())
			require.Equal(t, "mw2", usedMiddlewares[1].(MiddlewareName).MiddlewareName())
			require.Equal(t, "mw1", usedMiddlewares[2].(MiddlewareName).MiddlewareName())
		})

		t.Run("New client should verify that middlewares are not duplicated", func(t *testing.T) {
			ctx := &testContext{}
			_, err := New(Options{
				Middlewares: []Middleware{ctx.createMiddleware("mw1"), ctx.createMiddleware("mw1")},
			})
			require.ErrorContains(t, err, "middleware with name mw1 already exists")
		})

		t.Run("New client should verify that middlewares are not duplicated when configured", func(t *testing.T) {
			ctx := &testContext{}
			_, err := New(Options{
				Middlewares: []Middleware{ctx.createMiddleware("mw1")},
				ConfigureMiddleware: func(_ Options, existingMiddleware []Middleware) []Middleware {
					return append(existingMiddleware, ctx.createMiddleware("mw1"))
				},
			})
			require.ErrorContains(t, err, "middleware with name mw1 already exists")
		})

		t.Run("When roundtrip should call expected middlewares", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://www.google.com", nil)
			require.NoError(t, err)
			res, err := client.Transport.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, res)
			if res.Body != nil {
				require.NoError(t, res.Body.Close())
			}
			require.Len(t, ctx.callChain, 6)
			require.ElementsMatch(t, []string{"before mw3", "before mw2", "before mw1", "after mw1", "after mw2", "after mw3"}, ctx.callChain)
		})
	})
}

func TestRoundTripperFromMiddlewares(t *testing.T) {
	t.Run("Without any middleware should call final roundTripper", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		rt, err := roundTripperFromMiddlewares(Options{}, nil, finalRoundTripper)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)
	})

	t.Run("With 3 middlewares should call middlewares in expected order before calling the final roundTripper", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		middlewares := []Middleware{ctx.createMiddleware("mw1"), ctx.createMiddleware("mw2"), ctx.createMiddleware("mw3")}
		rt, err := roundTripperFromMiddlewares(Options{}, middlewares, finalRoundTripper)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 7)
		require.ElementsMatch(t, []string{"before mw1", "before mw2", "before mw3", "final", "after mw1", "after mw2", "after mw3"}, ctx.callChain)
	})
}

type testContext struct {
	callChain []string
}

func (c *testContext) createRoundTripper(name string) http.RoundTripper {
	return RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		c.callChain = append(c.callChain, name)
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
}

func (c *testContext) createRoundTripperWithError(err error) http.RoundTripper {
	return RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{}, err
	})
}

func (c *testContext) createMiddleware(name string) Middleware {
	return NamedMiddlewareFunc(name, func(_ Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			c.callChain = append(c.callChain, fmt.Sprintf("before %s", name))
			res, err := next.RoundTrip(req)
			c.callChain = append(c.callChain, fmt.Sprintf("after %s", name))
			return res, err
		})
	})
}

func TestReverseMiddlewares(t *testing.T) {
	t.Run("Should reverse 1 middleware", func(t *testing.T) {
		tCtx := testContext{}
		middlewares := []Middleware{
			tCtx.createMiddleware("mw1"),
		}
		reversed := reverseMiddlewares(middlewares)
		require.Len(t, reversed, 1)
		require.Equal(t, "mw1", reversed[0].(MiddlewareName).MiddlewareName())
	})

	t.Run("Should reverse 2 middlewares", func(t *testing.T) {
		tCtx := testContext{}
		middlewares := []Middleware{
			tCtx.createMiddleware("mw1"),
			tCtx.createMiddleware("mw2"),
		}
		reversed := reverseMiddlewares(middlewares)
		require.Len(t, reversed, 2)
		require.Equal(t, "mw2", reversed[0].(MiddlewareName).MiddlewareName())
		require.Equal(t, "mw1", reversed[1].(MiddlewareName).MiddlewareName())
	})

	t.Run("Should reverse 3 middlewares", func(t *testing.T) {
		tCtx := testContext{}
		middlewares := []Middleware{
			tCtx.createMiddleware("mw1"),
			tCtx.createMiddleware("mw2"),
			tCtx.createMiddleware("mw3"),
		}
		reversed := reverseMiddlewares(middlewares)
		require.Len(t, reversed, 3)
		require.Equal(t, "mw3", reversed[0].(MiddlewareName).MiddlewareName())
		require.Equal(t, "mw2", reversed[1].(MiddlewareName).MiddlewareName())
		require.Equal(t, "mw1", reversed[2].(MiddlewareName).MiddlewareName())
	})

	t.Run("Should reverse 4 middlewares", func(t *testing.T) {
		tCtx := testContext{}
		middlewares := []Middleware{
			tCtx.createMiddleware("mw1"),
			tCtx.createMiddleware("mw2"),
			tCtx.createMiddleware("mw3"),
			tCtx.createMiddleware("mw4"),
		}
		reversed := reverseMiddlewares(middlewares)
		require.Len(t, reversed, 4)
		require.Equal(t, "mw4", reversed[0].(MiddlewareName).MiddlewareName())
		require.Equal(t, "mw3", reversed[1].(MiddlewareName).MiddlewareName())
		require.Equal(t, "mw2", reversed[2].(MiddlewareName).MiddlewareName())
		require.Equal(t, "mw1", reversed[3].(MiddlewareName).MiddlewareName())
	})
}

func TestDefaultTransport(t *testing.T) {
	t.Run("Transport returned from GetTransport() with no arguments is not http.DefaultTransport", func(t *testing.T) {
		transport, err := GetTransport()
		require.NoError(t, err)
		// This is essentially the same check added to secure_socks_proxy.go in
		// https://github.com/grafana/grafana-plugin-sdk-go/pull/1295; since that's
		// addressing the issue we're concerned with here, it should suffice.
		require.NotEqual(t, transport, http.DefaultTransport.(*http.Transport))
	})
}
