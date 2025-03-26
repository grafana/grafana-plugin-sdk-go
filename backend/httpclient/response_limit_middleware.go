package httpclient

import (
	"net/http"
	"os"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	ResponseLimitEnvVar = "GF_DATAPROXY_RESPONSE_LIMIT"
)

// ResponseLimitMiddlewareName is the middleware name used by ResponseLimitMiddleware.
const ResponseLimitMiddlewareName = "response-limit"

func ResponseLimitMiddleware(limit int64) Middleware {
	if limit <= 0 {
		envLimit, ok := os.LookupEnv(ResponseLimitEnvVar)
		if ok && envLimit != "" {
			limitInt, err := strconv.ParseInt(envLimit, 10, 64)
			if err == nil && limitInt > 0 {
				limit = limitInt
			}

			log.DefaultLogger.Error("failed to parse GF_DATAPROXY_RESPONSE_LIMIT", "error", err)
		}

	}

	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(_ Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
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
