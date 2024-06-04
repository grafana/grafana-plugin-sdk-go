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
	ds := NewTestDS(t)
	req, settings := setupHealthRequest()
	collector := &TestCollector{}
	wrapper := slo.NewMetricsWrapper(ds, settings, collector)

	res, err := wrapper.CheckHealth(context.Background(), req)

	assert.Equal(t, nil, err)
	assert.Equal(t, backend.HealthStatusOk, res.Status)
	assert.True(t, collector.duration > 0)
}

func TestQueryWithMetrics(t *testing.T) {
	ds := NewTestDS(t)
	req, settings := setupQueryRequest()
	collector := &TestCollector{}
	wrapper := slo.NewMetricsWrapper(ds, settings, collector)

	_, err := wrapper.QueryData(context.Background(), req)

	assert.Equal(t, nil, err)
	assert.True(t, collector.duration > 0)
}

func TestResourceWithMetrics(t *testing.T) {
	ds := NewTestDS(t)
	req, settings := setupResourceRequest()
	collector := &TestCollector{}
	wrapper := slo.NewMetricsWrapper(ds, settings, collector)

	err := wrapper.CallResource(context.Background(), req, nil)

	assert.Equal(t, nil, err)
	assert.True(t, collector.duration > 0)
}

func NewTestDS(t *testing.T) *TestDS {
	t.Helper()
	client, clientErr := slo.NewClient()
	assert.Equal(t, nil, clientErr)
	return &TestDS{
		client: client,
	}
}

type TestDS struct {
	client *http.Client
}

func (m TestDS) QueryData(ctx context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	err := callGet(ctx, m)
	if err != nil {
		return nil, err
	}
	return &backend.QueryDataResponse{}, nil
}

func (m TestDS) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	err := callGet(ctx, m)
	if err != nil {
		return nil, err
	}
	return &backend.CheckHealthResult{
		Status: backend.HealthStatusOk,
	}, nil
}

func (m TestDS) CallResource(ctx context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
	err := callGet(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func callGet(ctx context.Context, m TestDS) error {
	r, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/get", nil)
	if err != nil {
		return err
	}
	res, err := m.client.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func setupHealthRequest() (*backend.CheckHealthRequest, backend.DataSourceInstanceSettings) {
	settings := backend.DataSourceInstanceSettings{Name: "foo", UID: "uid", Type: "type", JSONData: []byte(`{}`), DecryptedSecureJSONData: map[string]string{}}
	return &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &settings,
		},
	}, settings
}

func setupQueryRequest() (*backend.QueryDataRequest, backend.DataSourceInstanceSettings) {
	settings := backend.DataSourceInstanceSettings{Name: "foo", UID: "uid", Type: "type", JSONData: []byte(`{}`), DecryptedSecureJSONData: map[string]string{}}
	return &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &settings,
		},
	}, settings
}

func setupResourceRequest() (*backend.CallResourceRequest, backend.DataSourceInstanceSettings) {
	settings := backend.DataSourceInstanceSettings{Name: "foo", UID: "uid", Type: "type", JSONData: []byte(`{}`), DecryptedSecureJSONData: map[string]string{}}
	return &backend.CallResourceRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &settings,
		},
	}, settings
}

type TestCollector struct {
	duration float64
}

func (c *TestCollector) WithEndpoint(_ slo.Endpoint) slo.Collector {
	return c
}

func (c *TestCollector) CollectDuration(_ slo.Source, _ slo.Status, _ int, duration float64) {
	c.duration = duration
}
