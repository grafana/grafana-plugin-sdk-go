package httpclient

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// ResponseLimitMiddlewareName is the middleware name used by ResponseLimitMiddleware.
const ResponseLimitMiddlewareName = "response-limit"

func ResponseLimitMiddleware(limit int64) Middleware {
	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(_ Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			// Try to get limit from context first, fall back to static limit
			if cfgLimit := config.GrafanaConfigFromContext(req.Context()).ResponseLimit(); cfgLimit > 0 {
				limit = cfgLimit
			}

			if limit <= 0 {
				return res, nil
			}

			if res != nil && res.StatusCode != http.StatusSwitchingProtocols {
				res.Body = MaxBytesReader(res.Body, limit)
			}

			return res, nil
		})
	})
}
