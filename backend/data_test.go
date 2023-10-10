package backend

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
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

	t.Run("Response proxy JSON marshalling", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr

		require.Nil(t, qdr.ResponseProxy())

		b, err := json.Marshal(qdr)
		require.NoError(t, err)

		str := string(b)
		require.Equal(t, `{"results":{"A":{"status":200,"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}}}`, str)

		var qdrNew QueryDataResponse
		err = json.Unmarshal(b, &qdrNew)
		require.NoError(t, err)

		require.NotNil(t, qdrNew.Responses)
		require.NotNil(t, qdrNew.ResponseProxy())
		jsonProxy, ok := qdrNew.proxy.(*jsonResponseProxy)
		require.True(t, ok)
		require.NotNil(t, jsonProxy)
		require.Len(t, jsonProxy.raw.data, len(b))
		responses, err := qdrNew.ResponseProxy().Responses()
		require.NoError(t, err)
		require.NotNil(t, responses)
	})
}

func testDataResponse(t *testing.T) DataResponse {
	t.Helper()

	frames := data.Frames{
		data.NewFrame("simple",
			data.NewField("time", nil, []time.Time{
				time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
				time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
			}),
			data.NewField("valid", nil, []bool{true, false}),
		),
		data.NewFrame("other",
			data.NewField("value", nil, []float64{1.0}),
		),
	}
	return DataResponse{
		Frames: frames,
	}
}
