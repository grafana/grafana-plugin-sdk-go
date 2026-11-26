package slo_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/slo/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckHealthWithMetrics(t *testing.T) {
	ds, err := test.NewDS(t)
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
	ds, err := test.NewDS(t)
	assert.Equal(t, nil, err)
	req := query()
	collector := &test.Collector{}
	wrapper := slo.NewMetricsWrapper(ds, *req.PluginContext.DataSourceInstanceSettings, collector)

	_, err = wrapper.QueryData(context.Background(), req)

	assert.Equal(t, nil, err)
	assert.True(t, collector.Duration > 0)
}

func TestResourceWithMetrics(t *testing.T) {
	ds, err := test.NewDS(t)
	assert.Equal(t, nil, err)
	req := resource()
	collector := &test.Collector{}
	wrapper := slo.NewMetricsWrapper(ds, *req.PluginContext.DataSourceInstanceSettings, collector)

	err = wrapper.CallResource(context.Background(), req, nil)

	assert.Equal(t, nil, err)
	assert.True(t, collector.Duration > 0)
}

func TestNilCollectorFallsBackToDefault(t *testing.T) {
	ds, err := test.NewDS(t)
	require.NoError(t, err)
	req := query()
	wrapper := slo.NewMetricsWrapper(ds, *req.PluginContext.DataSourceInstanceSettings, nil)

	require.NotNil(t, wrapper.Metrics)
	assert.NotPanics(t, func() {
		_, err = wrapper.QueryData(context.Background(), req)
	})
	assert.NoError(t, err)
}

// disposablePlugin records whether Dispose was called
type disposablePlugin struct {
	disposed bool
}

func (p *disposablePlugin) Dispose() {
	p.disposed = true
}

func TestMetricsWrapperForwardsDispose(t *testing.T) {
	inner := &disposablePlugin{}
	wrapper := slo.NewMetricsWrapper(inner, backend.DataSourceInstanceSettings{Name: "foo", Type: "type"})

	// The instance manager only calls Dispose on instances that implement InstanceDisposer
	disposer, ok := any(wrapper).(instancemgmt.InstanceDisposer)
	require.True(t, ok)

	disposer.Dispose()
	assert.True(t, inner.disposed)
}

func TestMetricsWrapperDisposeWithoutInnerDisposer(t *testing.T) {
	wrapper := slo.NewMetricsWrapper(struct{}{}, backend.DataSourceInstanceSettings{Name: "foo", Type: "type"})

	disposer, ok := any(wrapper).(instancemgmt.InstanceDisposer)
	require.True(t, ok)

	assert.NotPanics(t, func() {
		disposer.Dispose()
	})
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
