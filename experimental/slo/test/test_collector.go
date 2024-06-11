package test

import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
)

type Collector struct {
	Duration float64
}

func (c *Collector) WithEndpoint(_ slo.Endpoint) slo.Collector {
	return c
}

func (c *Collector) CollectDuration(_ slo.Source, _ slo.Status, _ int, duration float64) {
	c.Duration = duration
}
