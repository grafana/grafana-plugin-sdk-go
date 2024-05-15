package slo

import (
	"errors"
	"net/http"
	"syscall"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "grafana",
	Name:      "plugin_external_requests_duration",
	Help:      "Duration of requests to external services",
}, []string{"plugin", "error_source"})

// Middleware captures duration of requests to external services and the source of errors
func Middleware(plugin string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(plugin, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripper(plugin, opts, next)
	})
}

// RoundTripper captures duration of requests to external services and the source of errors
func RoundTripper(plugin string, _ httpclient.Options, next http.RoundTripper) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		start := time.Now()
		var errorSource = "none"

		defer func() {
			duration.WithLabelValues(plugin, errorSource).Observe(time.Since(start).Seconds())
		}()

		res, err := next.RoundTrip(req)
		if res != nil && res.StatusCode >= 400 {
			errorSource = string(backend.ErrorSourceFromHTTPStatus(res.StatusCode))
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			errorSource = string(backend.ErrorSourceDownstream)
		}
		return res, err
	})
}
