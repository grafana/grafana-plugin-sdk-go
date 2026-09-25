package backend

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

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

func TestHeaderMiddlewareWithConfigureBasicAuthAfterContextualMiddleware(t *testing.T) {
	newClient := func(t *testing.T) *http.Client {
		t.Helper()

		client, err := httpclient.New(httpclient.Options{
			BasicAuth:           &httpclient.BasicAuthOptions{User: "user1", Password: "pwd"},
			ForwardHTTPHeaders:  true,
			ConfigureMiddleware: httpclient.ConfigureBasicAuthAfterContextualMiddleware,
		})
		require.NoError(t, err)
		return client
	}

	t.Run("A forwarded Authorization header wins over configured basic auth", func(t *testing.T) {
		received := runQueryDataThroughHeaderMiddleware(t, newClient(t), map[string]string{
			"Authorization": "Bearer 123",
		})

		require.Equal(t, "Bearer 123", received.Get("Authorization"))
	})

	t.Run("Without a forwarded Authorization header, falls back to configured basic auth", func(t *testing.T) {
		received := runQueryDataThroughHeaderMiddleware(t, newClient(t), nil)

		user, password, ok := (&http.Request{Header: received}).BasicAuth()
		require.True(t, ok)
		require.Equal(t, "user1", user)
		require.Equal(t, "pwd", password)
	})
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
