package slo

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics is an abstraction for collecting metrics
type Metrics struct {
	DSName   string
	DSType   string
	Endpoint Endpoint
}

// Duration is stored in the Context and used to collect metrics
type Duration struct {
	value      float64
	status     Status
	source     Source
	statusCode int
	mutex      sync.Mutex
}

func NewDuration(value float64) *Duration {
	return &Duration{value: value}
}

func (d *Duration) Add(value float64, source Source, statusCode int, err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.status == "" {
		d.status = "ok"
	}
	if err != nil {
		d.status = "error"
	}
	if statusCode >= 400 {
		d.status = "error"
	}

	// If the status code is now ok, but the previous status code was 401 or 403, mark it as ok
	// assuming a successful re-authentication ( token refresh, etc )
	if statusCode < 400 && (d.statusCode == 401 || d.statusCode == 403) {
		d.status = "ok"
	}

	d.statusCode = statusCode
	d.source = source
	d.value += value
}

func (d *Duration) Value() float64 {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.value
}

// Status is the status of the request
type Status string

// Endpoint is the endpoint of the request (health, query, resource)
type Endpoint string

// Source is the source of the error (downstream, plugin)
type Source string

const (
	StatusOK         Status   = "ok"
	StatusError      Status   = "error"
	EndpointHealth   Endpoint = "health"
	EndpointQuery    Endpoint = "query"
	EndpointResource Endpoint = "resource"
	SourceDownstream Source   = "downstream"
	SourcePlugin     Source   = "plugin"
)

var durationMetric = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "plugins",
	Name:      "plugin_request_duration_seconds",
	Help:      "Duration of plugin execution excluding any downstream call duration",
}, []string{"datasource_name", "datasource_type", "source", "endpoint", "status", "status_code"})

// NewMetrics creates a new Metrics instance
func NewMetrics(dsName, dsType string) Collector {
	dsName, ok := sanitizeLabelName(dsName)
	if !ok {
		backend.Logger.Warn("Failed to sanitize datasource name for prometheus label", dsName)
	}
	return Metrics{DSName: dsName, DSType: dsType}
}

// WithEndpoint returns a new Metrics instance with the given endpoint
func (m Metrics) WithEndpoint(endpoint Endpoint) Collector {
	return Metrics{DSName: m.DSName, DSType: m.DSType, Endpoint: endpoint}
}

// CollectDuration collects the duration as a metric
func (m Metrics) CollectDuration(source Source, status Status, statusCode int, duration float64) {
	durationMetric.WithLabelValues(m.DSName, m.DSType, string(source), string(m.Endpoint), string(status), fmt.Sprint(statusCode)).Observe(duration)
}

// SanitizeLabelName removes all invalid chars from the label name.
// If the label name is empty or contains only invalid chars, it will return false indicating it was not sanitized.
// copied from https://github.com/grafana/grafana/blob/main/pkg/infra/metrics/metricutil/utils.go#L14
func sanitizeLabelName(name string) (string, bool) {
	if len(name) == 0 {
		backend.Logger.Warn(fmt.Sprintf("label name cannot be empty: %s", name))
		return "", false
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
		backend.Logger.Warn(fmt.Sprintf("label name only contains invalid chars: %q", name))
		return "", false
	}

	return out.String(), true
}

// DurationKey is a key for storing the duration in the context
type DurationKey struct{}

// MetricsWrapper is a wrapper for a plugin that collects metrics
type MetricsWrapper struct {
	Name               string
	Type               string
	healthcheckHandler backend.CheckHealthHandler
	queryDataHandler   backend.QueryDataHandler
	resourceHandler    backend.CallResourceHandler
	Metrics            Collector
}

// NewMetricsWrapper creates a new MetricsWrapper instance
func NewMetricsWrapper(plugin any, s backend.DataSourceInstanceSettings, c ...Collector) *MetricsWrapper {
	collector := NewMetrics(s.Name, s.Type)
	if len(c) > 0 {
		collector = c[0]
	}
	wrapper := &MetricsWrapper{
		Name:    s.Name,
		Type:    s.Type,
		Metrics: collector,
	}
	if h, ok := plugin.(backend.CheckHealthHandler); ok {
		wrapper.healthcheckHandler = h
	}
	if q, ok := plugin.(backend.QueryDataHandler); ok {
		wrapper.queryDataHandler = q
	}
	if r, ok := plugin.(backend.CallResourceHandler); ok {
		wrapper.resourceHandler = r
	}
	return wrapper
}

// QueryData calls the QueryDataHandler and collects metrics
func (ds *MetricsWrapper) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	ctx = context.WithValue(ctx, DurationKey{}, &Duration{value: 0})
	metrics := ds.Metrics.WithEndpoint(EndpointQuery)

	start := time.Now()

	defer func() {
		collectDuration(ctx, start, metrics)
	}()

	return ds.queryDataHandler.QueryData(ctx, req)
}

// CheckHealth calls the CheckHealthHandler and collects metrics
func (ds *MetricsWrapper) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	ctx = context.WithValue(ctx, DurationKey{}, &Duration{value: 0})
	metrics := ds.Metrics.WithEndpoint(EndpointHealth)

	start := time.Now()

	defer func() {
		collectDuration(ctx, start, metrics)
	}()

	return ds.healthcheckHandler.CheckHealth(ctx, req)
}

// CallResource calls the CallResourceHandler and collects metrics
func (ds *MetricsWrapper) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	ctx = context.WithValue(ctx, DurationKey{}, &Duration{value: 0})
	metrics := ds.Metrics.WithEndpoint(EndpointResource)

	start := time.Now()

	defer func() {
		collectDuration(ctx, start, metrics)
	}()

	return ds.resourceHandler.CallResource(ctx, req, sender)
}

func collectDuration(ctx context.Context, start time.Time, metrics Collector) {
	totalDuration := time.Since(start).Seconds()
	downstreamDuration := ctx.Value(DurationKey{})
	if downstreamDuration != nil {
		d := downstreamDuration.(*Duration)
		pluginDuration := totalDuration - d.value
		metrics.CollectDuration(d.source, d.status, d.statusCode, pluginDuration)
	}
}

func SanitizeLabelName(name string) (string, error) {
	s, ok := sanitizeLabelName(name)
	if ok {
		return s, nil
	}
	return "", fmt.Errorf("failed to sanitize label name %s", name)
}

type Collector interface {
	CollectDuration(source Source, status Status, statusCode int, duration float64)
	WithEndpoint(endpoint Endpoint) Collector
}
