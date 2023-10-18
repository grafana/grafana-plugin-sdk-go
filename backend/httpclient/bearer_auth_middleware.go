package httpclient

import (
	"net/http"
)

// BearerAuthenticationMiddlewareName is the middleware name used by BearerAuthenticationMiddleware.
const BearerAuthenticationMiddlewareName = "BearerAuth"

// BearerAuthenticationMiddleware applies Bearer authentication to the HTTP header "Authorization"
// in the outgoing request.
// If Authorization header is already set, it will not be overridden by this middleware.
// If opts.BearerAuth is nil, next will be returned.
func BearerAuthenticationMiddleware() Middleware {
	return NamedMiddlewareFunc(BearerAuthenticationMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		if opts.BearerAuth == nil {
			return next
		}

		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") == "" {
				req.Header.Set("Authorization", "Bearer "+opts.BearerAuth.Token)
			}
			return next.RoundTrip(req)
		})
	})
}
