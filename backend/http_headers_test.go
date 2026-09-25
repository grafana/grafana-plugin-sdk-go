package backend

import (
	"bytes"
	"context"
	"io"
	"maps"
	"net/http"
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

func TestHeaderMiddlewareQueryChunkedData(t *testing.T) {
	const headerName = "X-Test-Header"
	request := &QueryChunkedDataRequest{}
	request.SetHTTPHeader(headerName, "test-value")

	var handlerCtx context.Context
	handler, err := HandlerFromMiddlewares(Handlers{
		QueryChunkedDataHandler: QueryChunkedDataHandlerFunc(func(ctx context.Context, _ *QueryChunkedDataRequest, _ ChunkedDataWriter) error {
			handlerCtx = ctx
			return nil
		}),
	}, newHeaderMiddleware())
	require.NoError(t, err)

	require.NoError(t, handler.QueryChunkedData(context.Background(), request, nil))

	middlewares := httpclient.ContextualMiddlewareFromContext(handlerCtx)
	require.Len(t, middlewares, 1)

	outgoingRequest, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	response, err := middlewares[0].CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: true}, httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Request:    req,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})).RoundTrip(outgoingRequest)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, "test-value", outgoingRequest.Header.Get(headerName))
}
