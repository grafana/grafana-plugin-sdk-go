package backend

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// forwardedCookiesMiddleware middleware that sets Cookie header on the
// outgoing request, if forwarded cookies configured/provided.
func forwardedCookiesMiddleware(headers map[string]string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc("forwarded-cookies", func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		forwardedCookies := forwardCookiesFromHTTPClientOptions(opts)
		if len(forwardedCookies) == 0 {
			return next
		}

		rawCookie, exists := headers["Cookie"]
		if !exists {
			return next
		}

		return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Cookie", rawCookie)

			return next.RoundTrip(req)
		})
	})
}

// forwardedOAuthIdentityMiddleware middleware that sets Authorization/X-ID-Token
// headers on the outgoing request, if forwarded OAuth identity configured/provided.
func forwardedOAuthIdentityMiddleware(headers map[string]string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc("forwarded-oauth-identity", func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		forwardOAuthIdentity := forwardOAuthIdentityFromHTTPClientOptions(opts)
		if !forwardOAuthIdentity {
			return next
		}

		authzHeader, authzHeaderExists := headers["Authorization"]
		idTokenHeader, idTokenHeaderExists := headers["X-ID-Token"]

		if !authzHeaderExists && !idTokenHeaderExists {
			return next
		}

		return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if authzHeaderExists {
				req.Header.Set("Authorization", authzHeader)
			}

			if idTokenHeaderExists {
				req.Header.Set("X-ID-Token", idTokenHeader)
			}

			return next.RoundTrip(req)
		})
	})
}
