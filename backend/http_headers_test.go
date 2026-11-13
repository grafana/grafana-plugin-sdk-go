package backend

import (
	"context"
	"io"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
	"github.com/stretchr/testify/require"
)

func TestHeaderMiddlewareCallResource(t *testing.T) {
	for _, tc := range []struct {
		name    string
		keyCase func(string) string
	}{
		{name: "canonical", keyCase: http.CanonicalHeaderKey},
		{name: "lowercase", keyCase: strings.ToLower},
		{name: "uppercase", keyCase: strings.ToUpper},
		{name: "mixed case", keyCase: func(s string) string { return strings.ToUpper(s[:1]) + strings.ToLower(s[1:]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, forward := range []bool{true, false} {
				name := "forwarding enabled"
				if !forward {
					name = "forwarding disabled"
				}
				t.Run(name, func(t *testing.T) {
					incoming := &CallResourceRequest{PluginContext: PluginContext{GrafanaConfig: config.NewGrafanaCfg(map[string]string{
						featuretoggles.EnabledFeatures: featuretoggles.PluginsFilterForwardedHeaders,
					})}, Headers: map[string][]string{
						tc.keyCase("Accept-Encoding"): {"zstd", "br"},
						tc.keyCase("X-Grafana-Id"):    {"synthetic-sign-in", "second-sign-in"},
						tc.keyCase("Authorization"):   {"Bearer synthetic-access"},
						tc.keyCase("X-Id-Token"):      {"synthetic-id"},
						tc.keyCase("Cookie"):          {"synthetic=cookie"},
						tc.keyCase("Content-Type"):    {"application/json"},
						tc.keyCase("Accept"):          {"application/json"},
						tc.keyCase("X-Custom"):        {"first", "second"},
					}}
					original := http.Header(incoming.Headers).Clone()
					handler := Handlers{CallResourceHandler: CallResourceHandlerFunc(func(ctx context.Context, req *CallResourceRequest, _ CallResourceResponseSender) error {
						require.Same(t, incoming, req)
						require.Equal(t, []string{"zstd", "br"}, req.GetHTTPHeaders().Values("Accept-Encoding"))
						require.Equal(t, []string{"synthetic-sign-in", "second-sign-in"}, req.GetHTTPHeaders().Values("X-Grafana-Id"))
						for _, explicit := range []bool{false, true} {
							outgoing, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.invalid/resource", nil)
							require.NoError(t, err)
							if explicit {
								outgoing.Header.Set("Accept-Encoding", "gzip")
								outgoing.Header.Set("X-Grafana-Id", "explicit-sign-in")
								outgoing.Header.Set("X-Custom", "explicit-custom")
							}
							transport := httpclient.ContextualMiddleware().CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: forward}, httpclient.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
								if explicit {
									require.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))
									require.Equal(t, "explicit-sign-in", r.Header.Get("X-Grafana-Id"))
									require.Equal(t, []string{"explicit-custom"}, r.Header.Values("X-Custom"))
								} else {
									require.NotContains(t, r.Header, "Accept-Encoding")
									require.NotContains(t, r.Header, "X-Grafana-Id")
									if forward {
										require.Equal(t, []string{"first", "second"}, r.Header.Values("X-Custom"))
									} else {
										require.Empty(t, r.Header)
									}
								}
								for _, key := range []string{"Authorization", "X-Id-Token", "Cookie", "Content-Type", "Accept"} {
									if forward {
										require.Equal(t, req.GetHTTPHeaders().Values(key), r.Header.Values(key))
									} else {
										require.NotContains(t, r.Header, key)
									}
								}
								return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
							}))
							resp, err := transport.RoundTrip(outgoing)
							require.NoError(t, err)
							require.NoError(t, resp.Body.Close())
						}
						return nil
					})}
					middleware, err := HandlerFromMiddlewares(handler, newHeaderMiddleware())
					require.NoError(t, err)
					sender := CallResourceResponseSenderFunc(func(*CallResourceResponse) error { return nil })
					require.NoError(t, middleware.CallResource(context.Background(), incoming, sender))
					require.Equal(t, original, http.Header(incoming.Headers))
				})
			}
		})
	}
}

func TestHeaderMiddlewareCallResourceNilRequest(t *testing.T) {
	called := false
	handler := Handlers{CallResourceHandler: CallResourceHandlerFunc(func(ctx context.Context, req *CallResourceRequest, _ CallResourceResponseSender) error {
		called = true
		require.Nil(t, req)
		require.Empty(t, httpclient.ContextualMiddlewareFromContext(ctx))
		return nil
	})}
	middleware := newHeaderMiddleware().CreateHandlerMiddleware(handler)
	require.NoError(t, middleware.CallResource(context.Background(), nil, nil))
	require.True(t, called)
}

func TestHeaderMiddlewareCallResourceConfiguredHeaders(t *testing.T) {
	called := false
	handler := Handlers{CallResourceHandler: CallResourceHandlerFunc(func(ctx context.Context, _ *CallResourceRequest, _ CallResourceResponseSender) error {
		client, err := httpclient.New(httpclient.Options{
			ForwardHTTPHeaders: true,
			Header:             http.Header{"Accept-Encoding": {"gzip"}, "X-Grafana-Id": {"explicit-sign-in"}},
			Middlewares: []httpclient.Middleware{
				httpclient.CustomHeadersMiddleware(),
				httpclient.ContextualMiddleware(),
				httpclient.MiddlewareFunc(func(_ httpclient.Options, _ http.RoundTripper) http.RoundTripper {
					return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
						called = true
						require.Equal(t, "gzip", req.Header.Get("Accept-Encoding"))
						require.Equal(t, "explicit-sign-in", req.Header.Get("X-Grafana-Id"))
						require.Equal(t, "Bearer synthetic-access", req.Header.Get("Authorization"))
						return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
					})
				}),
			},
		})
		require.NoError(t, err)
		outgoing, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.invalid/resource", nil)
		require.NoError(t, err)
		resp, err := client.Do(outgoing)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return nil
	})}
	middleware := newHeaderMiddleware().CreateHandlerMiddleware(handler)
	ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
		featuretoggles.EnabledFeatures: featuretoggles.PluginsFilterForwardedHeaders,
	}))
	require.NoError(t, middleware.CallResource(ctx, &CallResourceRequest{Headers: map[string][]string{
		"accept-encoding": {"zstd", "br"},
		"x-grafana-id":    {"synthetic-sign-in"},
		"authorization":   {"Bearer synthetic-access"},
	}}, nil))
	require.True(t, called)
}

func TestHeaderMiddlewareFilteringToggle(t *testing.T) {
	for _, endpoint := range []string{"resource", "query", "health"} {
		for _, tc := range []struct {
			name    string
			enabled bool
			forward bool
		}{
			{name: "toggle disabled", forward: true},
			{name: "toggle enabled", enabled: true, forward: true},
			{name: "toggle disabled forwarding disabled"},
			{name: "toggle enabled forwarding disabled", enabled: true},
		} {
			t.Run(endpoint+"/"+tc.name, func(t *testing.T) {
				cfg := config.NewGrafanaCfg(nil)
				if tc.enabled {
					cfg = config.NewGrafanaCfg(map[string]string{
						featuretoggles.EnabledFeatures: featuretoggles.PluginsFilterForwardedHeaders,
					})
				}
				pluginCtx := PluginContext{GrafanaConfig: cfg}
				incoming := http.Header{
					"Accept-Encoding": {"zstd"},
					"X-Grafana-Id":    {"synthetic-sign-in"},
					"Authorization":   {"Bearer synthetic-access"},
					"X-Id-Token":      {"synthetic-id"},
					"Cookie":          {"synthetic=cookie"},
					"Content-Type":    {"application/json"},
					"Accept":          {"application/json"},
					"X-Custom":        {"custom"},
				}
				// Query and health requests use the http_ prefix for arbitrary headers.
				stringHeaders := make(map[string]string, len(incoming))
				for key, values := range incoming {
					stringHeaders[httpHeaderPrefix+strings.ToLower(key)] = values[0]
				}
				original := maps.Clone(stringHeaders)
				called := false
				checkOutgoing := func(ctx context.Context, req ForwardHTTPHeaders) {
					called = true
					require.Equal(t, incoming, req.GetHTTPHeaders())
					var forwarded http.Header
					transport := httpclient.ContextualMiddleware().CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: tc.forward}, httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
						forwarded = req.Header.Clone()
						return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
					}))
					outgoing, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.invalid/resource", nil)
					require.NoError(t, err)
					resp, err := transport.RoundTrip(outgoing)
					require.NoError(t, err)
					require.NoError(t, resp.Body.Close())
					if !tc.forward {
						require.Empty(t, forwarded)
						return
					}
					if tc.enabled {
						require.NotContains(t, forwarded, "Accept-Encoding")
						require.NotContains(t, forwarded, "X-Grafana-Id")
					} else {
						require.Equal(t, "zstd", forwarded.Get("Accept-Encoding"))
						require.Equal(t, "synthetic-sign-in", forwarded.Get("X-Grafana-Id"))
					}
					for _, key := range []string{"Authorization", "X-Id-Token", "Cookie", "Content-Type", "Accept", "X-Custom"} {
						require.Equal(t, incoming.Values(key), forwarded.Values(key))
					}
				}
				handlers := Handlers{
					CallResourceHandler: CallResourceHandlerFunc(func(ctx context.Context, req *CallResourceRequest, _ CallResourceResponseSender) error {
						checkOutgoing(ctx, req)
						return nil
					}),
					QueryDataHandler: QueryDataHandlerFunc(func(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
						checkOutgoing(ctx, req)
						return &QueryDataResponse{}, nil
					}),
					CheckHealthHandler: CheckHealthHandlerFunc(func(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
						checkOutgoing(ctx, req)
						return &CheckHealthResult{}, nil
					}),
				}
				middleware, err := HandlerFromMiddlewares(handlers, newHeaderMiddleware())
				require.NoError(t, err)
				switch endpoint {
				case "resource":
					req := &CallResourceRequest{PluginContext: pluginCtx, Headers: map[string][]string(incoming.Clone())}
					sender := CallResourceResponseSenderFunc(func(*CallResourceResponse) error { return nil })
					require.NoError(t, middleware.CallResource(context.Background(), req, sender))
					require.Equal(t, incoming, http.Header(req.Headers))
				case "query":
					_, err = middleware.QueryData(context.Background(), &QueryDataRequest{PluginContext: pluginCtx, Headers: stringHeaders})
					require.NoError(t, err)
				case "health":
					_, err = middleware.CheckHealth(context.Background(), &CheckHealthRequest{PluginContext: pluginCtx, Headers: stringHeaders})
					require.NoError(t, err)
				}
				require.True(t, called)
				require.Equal(t, original, stringHeaders)
			})
		}
	}
}

func TestSetHTTPHeaderInStringMap(t *testing.T) {
	tcs := []struct {
		input    map[string]string
		expected map[string]string
	}{
		{
			expected: map[string]string{
				"":  "",
				"a": "",
			},
		},
		{
			input: map[string]string{
				"authorization": "a",
				"x-id-token":    "b",
				"cookie":        "c",
				"x-custom":      "d",
			},
			expected: map[string]string{
				"":              "",
				"a":             "",
				"authorization": "a",
				"Authorization": "a",
				"x-id-token":    "b",
				"X-Id-Token":    "b",
				"cookie":        "c",
				"Cookie":        "c",
				"x-custom":      "d",
				"X-Custom":      "d",
			},
		},
		{
			input: map[string]string{
				"Authorization": "a",
				"X-ID-Token":    "b",
				"Cookie":        "c",
				"X-Custom":      "d",
			},
			expected: map[string]string{
				"":              "",
				"a":             "",
				"authorization": "a",
				"Authorization": "a",
				"x-id-token":    "b",
				"X-Id-Token":    "b",
				"cookie":        "c",
				"Cookie":        "c",
				"x-custom":      "d",
				"X-Custom":      "d",
			},
		},
	}

	for _, tc := range tcs {
		headerMap := map[string]string{}
		for k, v := range tc.input {
			setHTTPHeaderInStringMap(headerMap, k, v)
		}
		headers := getHTTPHeadersFromStringMap(headerMap)

		for k, v := range tc.expected {
			require.Equal(t, v, headers.Get(k))
		}
	}
}

func TestGetHTTPHeadersFromStringMap(t *testing.T) {
	tcs := []struct {
		input    map[string]string
		expected map[string]string
	}{
		{
			expected: map[string]string{
				"":  "",
				"a": "",
			},
		},
		{
			input: map[string]string{
				"authorization":               "a",
				"x-id-token":                  "b",
				"cookie":                      "c",
				httpHeaderPrefix + "x-custom": "d",
			},
			expected: map[string]string{
				"":              "",
				"a":             "",
				"authorization": "a",
				"Authorization": "a",
				"x-id-token":    "b",
				"X-Id-Token":    "b",
				"cookie":        "c",
				"Cookie":        "c",
				"x-custom":      "d",
				"X-Custom":      "d",
			},
		},
		{
			input: map[string]string{
				"Authorization":               "a",
				"X-ID-Token":                  "b",
				"Cookie":                      "c",
				httpHeaderPrefix + "X-Custom": "d",
			},
			expected: map[string]string{
				"":              "",
				"a":             "",
				"authorization": "a",
				"Authorization": "a",
				"x-id-token":    "b",
				"X-Id-Token":    "b",
				"cookie":        "c",
				"Cookie":        "c",
				"x-custom":      "d",
				"X-Custom":      "d",
			},
		},
	}

	for _, tc := range tcs {
		headers := getHTTPHeadersFromStringMap(tc.input)

		for k, v := range tc.expected {
			require.Equal(t, v, headers.Get(k))
		}
	}
}

func TestDeleteHTTPHeaderInStringMap(t *testing.T) {
	tcs := []struct {
		input      map[string]string
		deleteKeys []string
		expected   map[string]string
	}{
		{
			expected: map[string]string{
				"":  "",
				"a": "",
			},
		},
		{
			input: map[string]string{
				"authorization":               "a",
				"x-id-token":                  "b",
				"cookie":                      "c",
				httpHeaderPrefix + "x-custom": "d",
			},
			deleteKeys: []string{"authorization", "x-id-token", "cookie", "x-custom"},
			expected: map[string]string{
				"":              "",
				"a":             "",
				"authorization": "",
				"Authorization": "",
				"x-id-token":    "",
				"X-Id-Token":    "",
				"cookie":        "",
				"Cookie":        "",
				"x-custom":      "",
				"X-Custom":      "",
			},
		},
		{
			input: map[string]string{
				"Authorization":               "a",
				"X-ID-Token":                  "b",
				"Cookie":                      "c",
				httpHeaderPrefix + "X-Custom": "d",
			},
			deleteKeys: []string{"Authorization", "X-Id-Token", "Cookie", "X-Custom"},
			expected: map[string]string{
				"":              "",
				"a":             "",
				"authorization": "",
				"Authorization": "",
				"x-id-token":    "",
				"X-Id-Token":    "",
				"cookie":        "",
				"Cookie":        "",
				"x-custom":      "",
				"X-Custom":      "",
			},
		},
	}

	for _, tc := range tcs {
		headerMap := make(map[string]string, len(tc.input))
		maps.Copy(headerMap, tc.input)

		for _, key := range tc.deleteKeys {
			deleteHTTPHeaderInStringMap(headerMap, key)
		}
		headers := getHTTPHeadersFromStringMap(headerMap)

		for k, v := range tc.expected {
			require.Equal(t, v, headers.Get(k))
		}
	}
}
