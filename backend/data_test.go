package backend

import (
	"errors"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/errorsource"

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

func TestResponse(t *testing.T) {
	for _, tc := range []struct {
		name            string
		err             error
		expStatus       errorsource.Status
		expErrorMessage string
		expErrorSource  errorsource.ErrorSource
	}{
		{
			name:            "generic error",
			err:             errors.New("other"),
			expStatus:       errorsource.StatusUnknown,
			expErrorMessage: "other",
			expErrorSource:  errorsource.ErrorSourcePlugin,
		},
		{
			name:            "downstream error",
			err:             errorsource.WithDownstreamSource(errors.New("bad gateway"), false),
			expStatus:       0,
			expErrorMessage: "bad gateway",
			expErrorSource:  errorsource.ErrorSourceDownstream,
		},
		{
			name:            "plugin error",
			err:             errorsource.WithPluginSource(errors.New("internal error"), false),
			expStatus:       0,
			expErrorMessage: "internal error",
			expErrorSource:  errorsource.ErrorSourcePlugin,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := ErrorResponse(tc.err)
			require.Error(t, res.Error)
			require.Equal(t, tc.expStatus, res.Status)
			require.Equal(t, tc.expErrorMessage, res.Error.Error())
			require.Equal(t, tc.expErrorSource, res.ErrorSource)
		})
	}
}

func TestResponseWithOptions(t *testing.T) {
	unknown := errorsource.New(errors.New("unknown"), errorsource.ErrorSourcePlugin, errorsource.StatusUnknown)
	badgateway := errorsource.New(errors.New("bad gateway"), errorsource.ErrorSourceDownstream, errorsource.StatusBadGateway)

	for _, tc := range []struct {
		name            string
		err             errorsource.Error
		expStatus       errorsource.Status
		expErrorMessage string
		expErrorSource  errorsource.ErrorSource
	}{
		{
			name:            "unknown error",
			err:             unknown,
			expStatus:       errorsource.StatusUnknown,
			expErrorMessage: unknown.Error(),
			expErrorSource:  errorsource.ErrorSourcePlugin,
		},
		{
			name:            "bad gateway",
			err:             badgateway,
			expStatus:       errorsource.StatusBadGateway,
			expErrorMessage: badgateway.Error(),
			expErrorSource:  errorsource.ErrorSourceDownstream,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := ErrorResponse(tc.err)
			require.Error(t, res.Error)
			require.Equal(t, tc.expStatus, res.Status)
			require.Equal(t, tc.expErrorMessage, res.Error.Error())
			require.Equal(t, tc.expErrorSource, res.ErrorSource)
		})
	}
}
