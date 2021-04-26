package httpclient

import (
	"encoding/base64"
	"net/http"
)

// BasicAuthenticationMiddlewareName the middleware name used by BasicAuthenticationMiddleware.
const BasicAuthenticationMiddlewareName = "BasicAuth"

// BasicAuthenticationMiddleware applies basic authentication to the HTTP header "Authentication"
// in the outgoing request.
//
// If opts.BasicAuth is nil, next will be returned.
func BasicAuthenticationMiddleware() Middleware {
	return NamedMiddlewareFunc(BasicAuthenticationMiddlewareName, func(opts *Options, next http.RoundTripper) http.RoundTripper {
		if opts == nil || opts.BasicAuth == nil {
			return next
		}

		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Authentication", getBasicAuthHeader(opts.BasicAuth.User, opts.BasicAuth.Password))

			return next.RoundTrip(req)
		})
	})
}

// getBasicAuthHeader returns a base64 encoded string from user and password.
func getBasicAuthHeader(user string, password string) string {
	var userAndPass = user + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(userAndPass))
}
