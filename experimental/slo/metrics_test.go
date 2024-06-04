package slo_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
	"github.com/stretchr/testify/assert"
)

func TestCheckHealthWithMetrics(t *testing.T) {
	client, clientErr := slo.NewClient()
	assert.Equal(t, nil, clientErr)
	ds := TestDS{
		client: client,
	}
	req, settings := setupRequest()
	collector := &TestCollector{}
	wrapper := slo.NewMetricsWrapper(ds, settings, collector)

	res, err := wrapper.CheckHealth(context.Background(), req)

	assert.Equal(t, nil, err)
	assert.Equal(t, backend.HealthStatusOk, res.Status)
	assert.True(t, collector.duration > 0)
}

type TestDS struct {
	client *http.Client
}

func (m TestDS) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return nil, nil
}

func (m TestDS) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/get", nil)
	if err != nil {
		return nil, err
	}
	res, err := m.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return &backend.CheckHealthResult{
		Status: backend.HealthStatusOk,
	}, nil
}

func setupRequest() (*backend.CheckHealthRequest, backend.DataSourceInstanceSettings) {
	settings := backend.DataSourceInstanceSettings{Name: "foo", UID: "uid", Type: "type", JSONData: []byte(`{}`), DecryptedSecureJSONData: map[string]string{}}
	return &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &settings,
		},
	}, settings
}

type TestCollector struct {
	duration float64
}

func (c *TestCollector) WithEndpoint(endpoint slo.Endpoint) slo.Collector {
	return c
}

func (c *TestCollector) CollectDuration(_ slo.Source, _ slo.Status, _ int, duration float64) {
	c.duration = duration
}
