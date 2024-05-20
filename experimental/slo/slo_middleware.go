package slo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "plugins",
	Name:      "plugin_external_requests_duration_seconds",
	Help:      "Duration of requests to external services",
}, []string{"datasource_name", "datasource_type", "error_source"})

const DataSourceSLOMiddlewareName = "slo"

// Middleware captures duration of requests to external services and the source of errors
func Middleware(plugin string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(DataSourceSLOMiddlewareName, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripper(plugin, opts, next)
	})
}

// RoundTripper captures duration of requests to external services and the source of errors
func RoundTripper(plugin string, opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		start := time.Now()
		var errorSource = "none"

		defer func() {
			if opts.Labels == nil {
				return
			}

			datasourceName, exists := opts.Labels["datasource_name"]
			if !exists {
				return
			}

			datasourceLabelName, err := SanitizeLabelName(datasourceName)
			// if the datasource named cannot be turned into a prometheus
			// label we will skip instrumenting these metrics.
			if err != nil {
				return
			}

			datasourceType, exists := opts.Labels["datasource_type"]
			if !exists {
				return
			}
			duration.WithLabelValues(datasourceLabelName, datasourceType, errorSource).Observe(time.Since(start).Seconds())
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

// SanitizeLabelName removes all invalid chars from the label name.
// If the label name is empty or contains only invalid chars, it
// will return an error.
func SanitizeLabelName(name string) (string, error) {
	if len(name) == 0 {
		return "", errors.New("label name cannot be empty")
	}

	out := strings.Builder{}
	for i, b := range name {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || (b >= '0' && b <= '9' && i > 0) {
			out.WriteRune(b)
		} else if b == ' ' {
			out.WriteRune('_')
		}
	}

	if out.Len() == 0 {
		return "", fmt.Errorf("label name only contains invalid chars: %q", name)
	}

	return out.String(), nil
}
