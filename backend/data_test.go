package backend

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueryDataRequest(t *testing.T) {
	req := &QueryDataRequest{}
	const customHeaderName = "X-Custom"

	t.Run("Legacy headers", func(t *testing.T) {
		req.Headers = map[string]string{
			"Authorization":  "a",
			"X-ID-Token":     "b",
			"Cookie":         "c",
			customHeaderName: "d",
		}

		t.Run("GetHTTPHeaders canonical form", func(t *testing.T) {
			headers := req.GetHTTPHeaders()
			require.Equal(t, "a", headers.Get(OAuthIdentityTokenHeaderName))
			require.Equal(t, "b", headers.Get(OAuthIdentityIDTokenHeaderName))
			require.Equal(t, "c", headers.Get(CookiesHeaderName))
			require.Empty(t, headers.Get(customHeaderName))
		})

		t.Run("GetHTTPHeader canonical form", func(t *testing.T) {
			require.Equal(t, "a", req.GetHTTPHeader(OAuthIdentityTokenHeaderName))
			require.Equal(t, "b", req.GetHTTPHeader(OAuthIdentityIDTokenHeaderName))
			require.Equal(t, "c", req.GetHTTPHeader(CookiesHeaderName))
			require.Empty(t, req.GetHTTPHeader(customHeaderName))
		})

		t.Run("DeleteHTTPHeader canonical form", func(t *testing.T) {
			req.DeleteHTTPHeader(OAuthIdentityTokenHeaderName)
			req.DeleteHTTPHeader(OAuthIdentityIDTokenHeaderName)
			req.DeleteHTTPHeader(CookiesHeaderName)
			req.DeleteHTTPHeader(customHeaderName)
			require.Empty(t, req.Headers)
		})
	})

	t.Run("SetHTTPHeader canonical form", func(t *testing.T) {
		req.SetHTTPHeader(OAuthIdentityTokenHeaderName, "a")
		req.SetHTTPHeader(OAuthIdentityIDTokenHeaderName, "b")
		req.SetHTTPHeader(CookiesHeaderName, "c")
		req.SetHTTPHeader(customHeaderName, "d")

		t.Run("GetHTTPHeaders canonical form", func(t *testing.T) {
			headers := req.GetHTTPHeaders()
			require.Equal(t, "a", headers.Get(OAuthIdentityTokenHeaderName))
			require.Equal(t, "b", headers.Get(OAuthIdentityIDTokenHeaderName))
			require.Equal(t, "c", headers.Get(CookiesHeaderName))
			require.Equal(t, "d", headers.Get(customHeaderName))
		})

		t.Run("GetHTTPHeader canonical form", func(t *testing.T) {
			require.Equal(t, "a", req.GetHTTPHeader(OAuthIdentityTokenHeaderName))
			require.Equal(t, "b", req.GetHTTPHeader(OAuthIdentityIDTokenHeaderName))
			require.Equal(t, "c", req.GetHTTPHeader(CookiesHeaderName))
			require.Equal(t, "d", req.GetHTTPHeader(customHeaderName))
		})

		t.Run("DeleteHTTPHeader canonical form", func(t *testing.T) {
			req.DeleteHTTPHeader(OAuthIdentityTokenHeaderName)
			req.DeleteHTTPHeader(OAuthIdentityIDTokenHeaderName)
			req.DeleteHTTPHeader(CookiesHeaderName)
			req.DeleteHTTPHeader(customHeaderName)
			require.Empty(t, req.Headers)
		})
	})

	t.Run("SetHTTPHeader non-canonical form", func(t *testing.T) {
		req.SetHTTPHeader(strings.ToLower(OAuthIdentityTokenHeaderName), "a")
		req.SetHTTPHeader(strings.ToLower(OAuthIdentityIDTokenHeaderName), "b")
		req.SetHTTPHeader(strings.ToLower(CookiesHeaderName), "c")
		req.SetHTTPHeader(strings.ToLower(customHeaderName), "d")

		t.Run("GetHTTPHeaders non-canonical form", func(t *testing.T) {
			headers := req.GetHTTPHeaders()
			require.Equal(t, "a", headers.Get(strings.ToLower(OAuthIdentityTokenHeaderName)))
			require.Equal(t, "b", headers.Get(strings.ToLower(OAuthIdentityIDTokenHeaderName)))
			require.Equal(t, "c", headers.Get(strings.ToLower(CookiesHeaderName)))
			require.Equal(t, "d", headers.Get(strings.ToLower(customHeaderName)))
		})

		t.Run("GetHTTPHeader non-canonical form", func(t *testing.T) {
			require.Equal(t, "a", req.GetHTTPHeader(strings.ToLower(OAuthIdentityTokenHeaderName)))
			require.Equal(t, "b", req.GetHTTPHeader(strings.ToLower(OAuthIdentityIDTokenHeaderName)))
			require.Equal(t, "c", req.GetHTTPHeader(strings.ToLower(CookiesHeaderName)))
			require.Equal(t, "d", req.GetHTTPHeader(strings.ToLower(customHeaderName)))
		})

		t.Run("DeleteHTTPHeader non-canonical form", func(t *testing.T) {
			req.DeleteHTTPHeader(strings.ToLower(OAuthIdentityTokenHeaderName))
			req.DeleteHTTPHeader(strings.ToLower(OAuthIdentityIDTokenHeaderName))
			req.DeleteHTTPHeader(strings.ToLower(CookiesHeaderName))
			req.DeleteHTTPHeader(strings.ToLower(customHeaderName))
			require.Empty(t, req.Headers)
		})
	})
}

func TestBatchDataQueriesByTimeRange(t *testing.T) {
	start := time.Date(2024, time.November, 29, 0, 42, 34, 0, time.UTC)
	FiveMin := time.Date(2024, time.November, 29, 0, 47, 34, 0, time.UTC)
	TenMin := time.Date(2024, time.November, 29, 0, 52, 34, 0, time.UTC)
	loc := time.FixedZone("UTC+1", 1*60*60)
	FiveMinDifferentZone := time.Date(2024, time.November, 29, 1, 47, 34, 0, loc)
	testQueries := []DataQuery{
		{
			RefID:     "A",
			TimeRange: TimeRange{From: start, To: FiveMin},
		},
		{
			RefID:     "B",
			TimeRange: TimeRange{From: start, To: TenMin},
		},
		{
			RefID:     "C",
			TimeRange: TimeRange{From: start, To: FiveMinDifferentZone},
		},
	}
	result := BatchDataQueriesByTimeRange(testQueries)
	require.Equal(t, 2, len(result))
	var FiveMinQueries = result[0]
	var TenMinQueries = result[1]
	if len(result[0]) == 1 {
		TenMinQueries = result[0]
		FiveMinQueries = result[1]
	}

	require.Equal(t, 2, len(FiveMinQueries))
	require.Equal(t, "A", FiveMinQueries[0].RefID)
	require.Equal(t, "C", FiveMinQueries[1].RefID)

	require.Equal(t, 1, len(TenMinQueries))
	require.Equal(t, "B", TenMinQueries[0].RefID)
}
