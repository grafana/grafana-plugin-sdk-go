package backend

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	prom "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// CheckHealthHandler enables users to send health check
// requests to a backend plugin
type CheckHealthHandler interface {
	CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error)
}

// CheckHealthHandlerFunc is an adapter to allow the use of
// ordinary functions as backend.CheckHealthHandler. If f is a function
// with the appropriate signature, CheckHealthHandlerFunc(f) is a
// Handler that calls f.
type CheckHealthHandlerFunc func(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error)

// CheckHealth calls fn(ctx, req).
func (fn CheckHealthHandlerFunc) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	return fn(ctx, req)
}

// HealthStatus is the status of the plugin.
type HealthStatus int

const (
	// HealthStatusUnknown means the status of the plugin is unknown.
	HealthStatusUnknown HealthStatus = iota

	// HealthStatusOk means the status of the plugin is good.
	HealthStatusOk

	// HealthStatusError means the plugin is in an error state.
	HealthStatusError
)

var healthStatusNames = map[int]string{
	0: "UNKNOWN",
	1: "OK",
	2: "ERROR",
}

// String textual represntation of the status.
func (hs HealthStatus) String() string {
	s, exists := healthStatusNames[int(hs)]
	if exists {
		return s
	}
	return strconv.Itoa(int(hs))
}

// CheckHealthRequest contains the healthcheck request
type CheckHealthRequest struct {
	PluginContext PluginContext
}

// CheckHealthResult contains the healthcheck response
type CheckHealthResult struct {
	Status      HealthStatus
	Message     string
	JSONDetails []byte
}

// CollectMetricsHandler handles metric collection.
type CollectMetricsHandler interface {
	CollectMetrics(ctx context.Context) (*CollectMetricsResult, error)
}

// CollectMetricsHandlerFunc is an adapter to allow the use of
// ordinary functions as backend.CollectMetricsHandler. If f is a function
// with the appropriate signature, CollectMetricsHandlerFunc(f) is a
// Handler that calls f.
type CollectMetricsHandlerFunc func(ctx context.Context) (*CollectMetricsResult, error)

// CollectMetrics calls fn(ctx, req).
func (fn CollectMetricsHandlerFunc) CollectMetrics(ctx context.Context) (*CollectMetricsResult, error) {
	return fn(ctx)
}

// CollectMetricsResult collect metrics result.
type CollectMetricsResult struct {
	PrometheusMetrics []byte
}

// ToUsageStats filters a CollectMetricsResult and returns usage stats
func (res *CollectMetricsResult) ToUsageStats(pluginID string) map[string]interface{} {
	var (
		parser expfmt.TextParser
		r      = regexp.MustCompile(`usage_stats[_]?`)
		stats  = map[string]interface{}{}
		reader = bytes.NewReader(res.PrometheusMetrics)
	)

	mfs, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		Logger.Error("unable to parse usage stats from metric", "plugin", pluginID, "err", err)
	}
	for _, mf := range mfs {
		if !strings.HasPrefix(mf.GetName(), "usage_stats") {
			continue
		}
		name := r.ReplaceAllString(mf.GetName(), "")
		for _, m := range mf.Metric {
			labels := []string{pluginID, name}
			for _, l := range m.Label {
				labels = append(labels, l.GetName(), l.GetValue())
			}
			converted := map[string]interface{}{}
			switch mf.GetType() {
			case prom.MetricType_COUNTER:
				converted = convertCounter(labels, m)
			case prom.MetricType_GAUGE:
				converted = convertGauge(labels, m)
			case prom.MetricType_SUMMARY:
				converted = convertSummary(labels, m)
			case prom.MetricType_HISTOGRAM:
				converted = convertHistogram(labels, m)
			}
			for k, v := range converted {
				stats[k] = v
			}
		}
	}

	return stats
}

func convertCounter(labels []string, m *prom.Metric) map[string]interface{} {
	stats := map[string]interface{}{}
	key := toUsageStatKey(labels...)
	stats[key] = m.GetCounter().GetValue()
	return stats
}

func convertGauge(labels []string, m *prom.Metric) map[string]interface{} {
	stats := map[string]interface{}{}
	key := toUsageStatKey(labels...)
	stats[key] = m.GetGauge().GetValue()
	return stats
}

func convertSummary(labels []string, m *prom.Metric) map[string]interface{} {
	stats := map[string]interface{}{}
	summary := m.GetSummary()
	key := toUsageStatKey(append(labels, "sum")...)
	stats[key] = summary.GetSampleSum()
	key = toUsageStatKey(append(labels, "count")...)
	stats[key] = float64(summary.GetSampleCount())
	for _, q := range summary.Quantile {
		key = toUsageStatKey(append(labels, "quantile", fmt.Sprintf("%.2f", q.GetQuantile()))...)
		stats[key] = q.GetValue()
	}
	return stats
}

func convertHistogram(labels []string, m *prom.Metric) map[string]interface{} {
	h := m.GetHistogram()
	stats := map[string]interface{}{}
	key := toUsageStatKey(append(labels, "sum")...)
	stats[key] = h.GetSampleSum()
	key = toUsageStatKey(append(labels, "count")...)
	stats[key] = float64(h.GetSampleCount())
	for _, b := range h.Bucket {
		key = toUsageStatKey(append(labels, "bucket", fmt.Sprintf("%.0f", b.GetUpperBound()))...)
		stats[key] = float64(b.GetCumulativeCount())
	}
	return stats
}

func toUsageStatKey(s ...string) string {
	cleaned := make([]string, len(s))
	for i, v := range s {
		cleaned[i] = strings.ToLower(strings.ReplaceAll(v, ".", "_"))
	}
	return "stats.plugin." + strings.Join(cleaned, ".")
}
