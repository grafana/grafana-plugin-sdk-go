package httpclient

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	t.Run("NewProvider() without any options", func(t *testing.T) {
		provider := NewProvider()
		require.NotNil(t, provider)
		require.Equal(t, BasicAuthenticationMiddlewareName, provider.Opts.Middlewares[0].(MiddlewareName).MiddlewareName())
		require.Equal(t, CustomHeadersMiddlewareName, provider.Opts.Middlewares[1].(MiddlewareName).MiddlewareName())

		t.Run("New client should use default middlewares", func(t *testing.T) {
			usedMiddlewares := []Middleware{}
			client, err := provider.New(Options{
				ConfigureMiddleware: func(opts Options, existingMiddleware []Middleware) []Middleware {
					usedMiddlewares = make([]Middleware, len(existingMiddleware))
					copy(usedMiddlewares, existingMiddleware)
					return existingMiddleware
				},
			})
			require.NoError(t, err)
			require.NotNil(t, client)
			require.Len(t, usedMiddlewares, 2)
			require.Equal(t, BasicAuthenticationMiddlewareName, usedMiddlewares[0].(MiddlewareName).MiddlewareName())
			require.Equal(t, CustomHeadersMiddlewareName, usedMiddlewares[1].(MiddlewareName).MiddlewareName())
		})

		t.Run("Transport should use default middlewares", func(t *testing.T) {
			usedMiddlewares := []Middleware{}
			transport, err := provider.GetTransport(Options{
				ConfigureMiddleware: func(opts Options, existingMiddleware []Middleware) []Middleware {
					usedMiddlewares = make([]Middleware, len(existingMiddleware))
					copy(usedMiddlewares, existingMiddleware)
					return existingMiddleware
				},
			})
			require.NoError(t, err)
			require.NotNil(t, transport)
			require.Len(t, usedMiddlewares, 2)
			require.Equal(t, BasicAuthenticationMiddlewareName, usedMiddlewares[0].(MiddlewareName).MiddlewareName())
			require.Equal(t, CustomHeadersMiddlewareName, usedMiddlewares[1].(MiddlewareName).MiddlewareName())
		})

		t.Run("New() with options and no middleware should return expected http client and transport", func(t *testing.T) {
			client, err := provider.New(Options{
				Timeouts: &TimeoutOptions{
					Timeout:               time.Second,
					KeepAlive:             2 * time.Second,
					TLSHandshakeTimeout:   3 * time.Second,
					ExpectContinueTimeout: 4 * time.Second,
					MaxIdleConns:          5,
					MaxIdleConnsPerHost:   7,
					IdleConnTimeout:       6 * time.Second,
				},
				Middlewares: []Middleware{},
			})
			require.NoError(t, err)
			require.NotNil(t, client)
			require.Equal(t, time.Second, client.Timeout)

			transport, ok := client.Transport.(*http.Transport)
			require.True(t, ok)
			require.NotNil(t, transport)
			require.Equal(t, 3*time.Second, transport.TLSHandshakeTimeout)
			require.Equal(t, 4*time.Second, transport.ExpectContinueTimeout)
			require.Equal(t, 5, transport.MaxIdleConns)
			require.Equal(t, 7, transport.MaxIdleConnsPerHost)
			require.Equal(t, 6*time.Second, transport.IdleConnTimeout)
		})

		t.Run("New() with options middleware should return expected http.Client", func(t *testing.T) {
			ctx := &testContext{}
			usedMiddlewares := []Middleware{}
			client, err := provider.New(Options{
				Middlewares: []Middleware{ctx.createMiddleware("mw1"), ctx.createMiddleware("mw2"), ctx.createMiddleware("mw3")},
				ConfigureMiddleware: func(opts Options, existingMiddleware []Middleware) []Middleware {
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
				require.Equal(t, "mw1", usedMiddlewares[0].(MiddlewareName).MiddlewareName())
				require.Equal(t, "mw2", usedMiddlewares[1].(MiddlewareName).MiddlewareName())
				require.Equal(t, "mw3", usedMiddlewares[2].(MiddlewareName).MiddlewareName())
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
	})

	t.Run("NewProvider() with options", func(t *testing.T) {
		opts := ProviderOptions{
			Timeout: &TimeoutOptions{
				Timeout:               time.Second,
				KeepAlive:             2 * time.Second,
				TLSHandshakeTimeout:   3 * time.Second,
				ExpectContinueTimeout: 4 * time.Second,
				MaxIdleConns:          5,
				MaxIdleConnsPerHost:   7,
				IdleConnTimeout:       6 * time.Second,
			},
		}

		t.Run("Should use provider options when calling New() without options", func(t *testing.T) {
			ctx := newProviderTestContext(t, opts)
			client, err := ctx.provider.New()
			require.NoError(t, err)
			require.NotNil(t, client)
			require.Equal(t, time.Second, client.Timeout)

			require.Equal(t, 1, ctx.configureMiddlewareCount)
			require.Equal(t, 1, ctx.configureClientCount)
			require.Equal(t, 1, ctx.configureTransportCount)
			require.Equal(t, 1, ctx.configureTLSConfigCount)
		})

		t.Run("Should use provider options when calling GetTransport() without options", func(t *testing.T) {
			ctx := newProviderTestContext(t, opts)
			transport, err := ctx.provider.GetTransport()
			require.NoError(t, err)
			require.NotNil(t, transport)
			require.Equal(t, 3*time.Second, ctx.transport.TLSHandshakeTimeout)
			require.Equal(t, 4*time.Second, ctx.transport.ExpectContinueTimeout)
			require.Equal(t, 5, ctx.transport.MaxIdleConns)
			require.Equal(t, 7, ctx.transport.MaxIdleConnsPerHost)
			require.Equal(t, 6*time.Second, ctx.transport.IdleConnTimeout)

			require.Equal(t, 1, ctx.configureMiddlewareCount)
			require.Equal(t, 0, ctx.configureClientCount)
			require.Equal(t, 1, ctx.configureTransportCount)
			require.Equal(t, 1, ctx.configureTLSConfigCount)
		})
	})
}

type providerTestContext struct {
	configureMiddlewareCount int
	configureClientCount     int
	configureTransportCount  int
	configureTLSConfigCount  int
	client                   *http.Client
	transport                *http.Transport
	tlsConfig                *tls.Config
	provider                 *Provider
}

func newProviderTestContext(t *testing.T, opts ...ProviderOptions) *providerTestContext {
	t.Helper()
	ctx := &providerTestContext{}

	var providerOpts ProviderOptions
	if len(opts) > 0 {
		providerOpts = opts[0]
	} else {
		providerOpts = ProviderOptions{}
	}

	providerOpts.ConfigureMiddleware = func(opts Options, existingMiddleware []Middleware) []Middleware {
		ctx.configureMiddlewareCount++
		return existingMiddleware
	}
	providerOpts.ConfigureClient = func(opts Options, client *http.Client) {
		ctx.configureClientCount++
		ctx.client = client
	}
	providerOpts.ConfigureTransport = func(opts Options, transport *http.Transport) {
		ctx.configureTransportCount++
		ctx.transport = transport
	}
	providerOpts.ConfigureTLSConfig = func(opts Options, config *tls.Config) {
		ctx.configureTLSConfigCount++
		ctx.tlsConfig = config
	}

	ctx.provider = NewProvider(providerOpts)

	return ctx
}
