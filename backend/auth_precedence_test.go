package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	basicAuthCreds   = &httpclient.BasicAuthOptions{User: "user1", Password: "pwd"}
	customAuthHeader = http.Header{"Authorization": {"custom-value"}}
	originalHeaders  = map[string]string{"Authorization": "Bearer 123"}
)

func receivedBasicAuthHeader(t *testing.T, received http.Header) {
	t.Helper()
	user, password, ok := (&http.Request{Header: received}).BasicAuth()
	require.True(t, ok)
	assert.Equal(t, "user1", user)
	assert.Equal(t, "pwd", password)
}

func receivedCustomAuthHeader(t *testing.T, received http.Header) {
	t.Helper()
	assert.Equal(t, "custom-value", received.Get("Authorization"))
}

func receivedForwardedAuthHeader(t *testing.T, received http.Header) {
	t.Helper()
	assert.Equal(t, "Bearer 123", received.Get("Authorization"))
}

type testCase struct {
	name            string
	opts            httpclient.Options
	originalHeaders map[string]string
	assert          func(t *testing.T, received http.Header)
}

// These tests establish a contract of auth precedence
func TestDefaultMiddlewaresAuthPrecedence(t *testing.T) {
	tests := []testCase{
		{
			name:            "With ForwardHTTPHeaders=true and basic auth, server receives basic auth header",
			opts:            httpclient.Options{ForwardHTTPHeaders: true, BasicAuth: basicAuthCreds},
			originalHeaders: originalHeaders,
			assert:          receivedBasicAuthHeader,
		},
		{
			name:            "With ForwardHTTPHeaders=true and basic auth and no header to forward, server receives basic auth header",
			opts:            httpclient.Options{ForwardHTTPHeaders: true, BasicAuth: basicAuthCreds},
			originalHeaders: nil,
			assert:          receivedBasicAuthHeader,
		},
		{
			name:            "With custom auth header and ForwardHTTPHeaders=true, server receives custom auth header",
			opts:            httpclient.Options{Header: customAuthHeader, ForwardHTTPHeaders: true},
			originalHeaders: originalHeaders,
			assert:          receivedCustomAuthHeader,
		},
		{
			name:            "With custom auth header and basic auth, server receives custom auth header",
			opts:            httpclient.Options{Header: customAuthHeader, BasicAuth: basicAuthCreds},
			originalHeaders: nil,
			assert:          receivedCustomAuthHeader,
		},
		{
			name:            "With custom auth header, ForwardHTTPHeaders=true, and basic auth, server receives custom auth header",
			opts:            httpclient.Options{Header: customAuthHeader, ForwardHTTPHeaders: true, BasicAuth: basicAuthCreds},
			originalHeaders: originalHeaders,
			assert:          receivedCustomAuthHeader,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := httpclient.New(tc.opts)
			require.NoError(t, err)
			received := runQueryDataThroughHeaderMiddleware(t, client, tc.originalHeaders)
			tc.assert(t, received)
		})
	}
}

func TestConfigureBasicAuthAfterContextualMiddlewareAuthPrecedence(t *testing.T) {
	withRestoredOrder := func(opts httpclient.Options) httpclient.Options {
		opts.ConfigureMiddleware = httpclient.ConfigureBasicAuthAfterContextualMiddleware
		return opts
	}

	tests := []testCase{
		{
			name:            "With ForwardHTTPHeaders=true and basic auth, server receives forwarded auth header",
			opts:            withRestoredOrder(httpclient.Options{ForwardHTTPHeaders: true, BasicAuth: basicAuthCreds}),
			originalHeaders: originalHeaders,
			assert:          receivedForwardedAuthHeader,
		},
		{
			name:            "With ForwardHTTPHeaders=true and basic auth and no header to forward, server receives basic auth header",
			opts:            withRestoredOrder(httpclient.Options{ForwardHTTPHeaders: true, BasicAuth: basicAuthCreds}),
			originalHeaders: nil,
			assert:          receivedBasicAuthHeader,
		},
		{
			name:            "With custom auth header and ForwardHTTPHeaders=true, server receives custom auth header",
			opts:            withRestoredOrder(httpclient.Options{Header: customAuthHeader, ForwardHTTPHeaders: true}),
			originalHeaders: originalHeaders,
			assert:          receivedCustomAuthHeader,
		},
		{
			name:            "With custom auth header and basic auth, server receives custom auth header",
			opts:            withRestoredOrder(httpclient.Options{Header: customAuthHeader, BasicAuth: basicAuthCreds}),
			originalHeaders: nil,
			assert:          receivedCustomAuthHeader,
		},
		{
			name:            "With custom auth header, ForwardHTTPHeaders=true, and basic auth, server receives custom auth header",
			opts:            withRestoredOrder(httpclient.Options{Header: customAuthHeader, ForwardHTTPHeaders: true, BasicAuth: basicAuthCreds}),
			originalHeaders: originalHeaders,
			assert:          receivedCustomAuthHeader,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := httpclient.New(tc.opts)
			require.NoError(t, err)
			received := runQueryDataThroughHeaderMiddleware(t, client, tc.originalHeaders)
			tc.assert(t, received)
		})
	}
}

// runQueryDataThroughHeaderMiddleware sends a QueryDataRequest carrying headers through
// NewHeaderMiddleware() and returns the headers the httptest.Server actually received.
func runQueryDataThroughHeaderMiddleware(t *testing.T, client *http.Client, headers map[string]string) http.Header {
	t.Helper()

	var received http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	handler := newHeaderMiddleware().CreateHandlerMiddleware(Handlers{
		QueryDataHandler: QueryDataHandlerFunc(func(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			res, err := client.Do(req)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			return &QueryDataResponse{}, nil
		}),
	})

	_, err := handler.QueryData(context.Background(), &QueryDataRequest{Headers: headers})
	require.NoError(t, err)

	return received
}
