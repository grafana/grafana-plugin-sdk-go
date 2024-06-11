package slo_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo/test"
	"github.com/stretchr/testify/assert"
)

func TestCheckHealthWithMetrics(t *testing.T) {
	ds, err := test.NewDS()
	assert.Equal(t, nil, err)
	req := health()
	collector := &test.Collector{}
	wrapper := slo.NewMetricsWrapper(ds, *req.PluginContext.DataSourceInstanceSettings, collector)

	res, err := wrapper.CheckHealth(context.Background(), req)

	assert.Equal(t, nil, err)
	assert.Equal(t, backend.HealthStatusOk, res.Status)
	assert.True(t, collector.Duration > 0)
}

func TestQueryWithMetrics(t *testing.T) {
	ds, err := test.NewDS()
	assert.Equal(t, nil, err)
	req := query()
	collector := &test.Collector{}
	wrapper := slo.NewMetricsWrapper(ds, *req.PluginContext.DataSourceInstanceSettings, collector)

	_, err = wrapper.QueryData(context.Background(), req)

	assert.Equal(t, nil, err)
	assert.True(t, collector.Duration > 0)
}

func TestResourceWithMetrics(t *testing.T) {
	ds, err := test.NewDS()
	assert.Equal(t, nil, err)
	req := resource()
	collector := &test.Collector{}
	wrapper := slo.NewMetricsWrapper(ds, *req.PluginContext.DataSourceInstanceSettings, collector)

	err = wrapper.CallResource(context.Background(), req, nil)

	assert.Equal(t, nil, err)
	assert.True(t, collector.Duration > 0)
}

func health() *backend.CheckHealthRequest {
	return &backend.CheckHealthRequest{
		PluginContext: pluginCtx(),
	}
}

func query() *backend.QueryDataRequest {
	return &backend.QueryDataRequest{
		PluginContext: pluginCtx(),
	}
}

func resource() *backend.CallResourceRequest {
	return &backend.CallResourceRequest{
		PluginContext: pluginCtx(),
	}
}

func pluginCtx() backend.PluginContext {
	return backend.PluginContext{
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
			Name:                    "foo",
			UID:                     "uid",
			Type:                    "type",
			JSONData:                []byte(`{}`),
			DecryptedSecureJSONData: map[string]string{},
		},
	}
}
