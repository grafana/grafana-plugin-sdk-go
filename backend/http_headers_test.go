package backend

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestSetHTTPHeaderInStringMapReplacesCaseInsensitively(t *testing.T) {
	headerMap := map[string]string{
		"Authorization": "raw",
	}

	setHTTPHeaderInStringMap(headerMap, "x-custom", "first")
	setHTTPHeaderInStringMap(headerMap, "X-Custom", "second")
	setHTTPHeaderInStringMap(headerMap, "authorization", "new")

	// Run repeatedly: before the fix the winner depended on map iteration order.
	for range 50 {
		headers := getHTTPHeadersFromStringMap(headerMap)
		require.Equal(t, "second", headers.Get("X-Custom"))
		require.Equal(t, "new", headers.Get("Authorization"))
	}
	require.Len(t, headerMap, 2)
}

func TestDeleteHTTPHeaderInStringMapRemovesAllSpellings(t *testing.T) {
	headerMap := map[string]string{
		"Authorization":      "raw",
		"http_authorization": "prefixed",
		"http_X-Custom":      "a",
		"http_x-custom":      "b",
		"http_X-Other":       "keep",
	}

	deleteHTTPHeaderInStringMap(headerMap, "Authorization")
	deleteHTTPHeaderInStringMap(headerMap, "x-custom")

	require.Equal(t, map[string]string{"http_X-Other": "keep"}, headerMap)
}
