package errorsource

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrorSourceMiddlewareName is the middleware name used by ErrorSourceMiddleware.
const ErrorSourceMiddlewareName = "ErrorSource"

var (
	errors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "plugins",
		Name:      "datasource_error_total",
		Help:      "Total number of times a plugin errored",
	}, []string{"plugin_name", "error_source"})
)

// ErrorSourceMiddleware captures error source metric
func ErrorSourcenMiddleware(plugin string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(plugin, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if res != nil && res.StatusCode >= 400 {
				errorSource := backend.GetErrorSource(res.StatusCode)
				errors.WithLabelValues(plugin, string(errorSource)).Inc()
			}
			return res, err
		})
	})
}
