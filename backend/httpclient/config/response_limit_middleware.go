package config

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// ResponseLimitMiddlewareName is the middleware name used by ResponseLimitMiddleware.
const ResponseLimitMiddlewareName = "response-limit"

func ResponseLimitMiddleware(limit int64) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			reqLimit := limit
			if reqLimit <= 0 {
				reqLimit = backend.GrafanaConfigFromContext(req.Context()).ResponseLimit()
			}

			if reqLimit <= 0 {
				return res, nil
			}

			if res != nil && res.StatusCode != http.StatusSwitchingProtocols {
				res.Body = MaxBytesReader(res.Body, reqLimit)
			}

			return res, nil
		})
	})
}
