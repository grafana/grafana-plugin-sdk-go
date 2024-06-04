package test

import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
)

type TestCollector struct {
	Duration float64
}

func (c *TestCollector) WithEndpoint(_ slo.Endpoint) slo.Collector {
	return c
}

func (c *TestCollector) CollectDuration(_ slo.Source, _ slo.Status, _ int, duration float64) {
	c.Duration = duration
}
