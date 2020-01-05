package backend

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

type ConfigureMetricsCollector interface {
	Register(cs ...prometheus.Collector) ConfigureMetricsCollector
}

type ConfigureResource interface {
	Resource(pattern string, fn func(ConfigureResource)) ConfigurePlugin
	Handle(pattern string, handler http.HandlerFunc) ConfigureResource
	Get(pattern string, handler http.HandlerFunc) ConfigureResource
	Update(pattern string, handler http.HandlerFunc) ConfigureResource
	Post(pattern string, handler http.HandlerFunc) ConfigureResource
	Delete(pattern string, handler http.HandlerFunc) ConfigureResource
}

type ConfigurePlugin interface {
	Metrics(fn func(ConfigureMetricsCollector)) ConfigurePlugin
	HealthCheck(CheckHealthHandler) ConfigurePlugin
	Resource(name string, fn func(ConfigureResource)) ConfigurePlugin
}

type PluginImpl struct {
	diagnostics *DiagnosticsGRPCPlugin
	backend     *CoreImpl
	datasource  *CoreImpl
	transform   *TransformImpl
}

type configureMetricsCollector struct {
	collectors []prometheus.Collector
}

func newMetricsCollectorConfigurer() ConfigureMetricsCollector {
	return &configureMetricsCollector{
		collectors: []prometheus.Collector{},
	}
}

func (c *configureMetricsCollector) Register(cs ...prometheus.Collector) ConfigureMetricsCollector {
	c.collectors = append(c.collectors, cs...)
	return c
}

type configurePlugin struct {
	metricsConfigurer ConfigureMetricsCollector
}

func newBackendPluginConfigurer() *configurePlugin {
	return &configurePlugin{
		metricsConfigurer: newMetricsCollectorConfigurer(),
	}
}

func newDatasourcePluginConfigurer() *configurePlugin {
	config := newBackendPluginConfigurer()
	return config
}

func (b *configurePlugin) Metrics(configureFn func(ConfigureMetricsCollector)) ConfigurePlugin {
	if configureFn == nil {
		panic("Metrics configure func cannot be nil")
	}

	configureFn(b.metricsConfigurer)
	return b
}

func (b *configurePlugin) HealthCheck(CheckHealthHandler) ConfigurePlugin {
	return b
}

func (b *configurePlugin) Resource(name string, fn func(resource ConfigureResource)) ConfigurePlugin {
	return b
}

func (b *configurePlugin) build() *PluginImpl {
	return nil
}

func BackendProvider(configure ConfigurePlugin) {
	configure.
		Metrics(func(m ConfigureMetricsCollector) {
			m.Register(nil)
		}).
		HealthCheck(nil).
		Resource("test", func(r ConfigureResource) {
			r.Get(":id", nil)
			r.Update(":id", nil)
			r.Post(":id", nil)
			r.Delete(":id", nil)
		}).
		Resource("test2", func(r ConfigureResource) {
			r.Get(":id", nil)
			r.Update(":id", nil)
			r.Post(":id", nil)
			r.Delete(":id", nil)
		})
}

func DatasourceProvider(configure ConfigurePlugin) QueryDataHandler {
	configure.
		Metrics(func(m ConfigureMetricsCollector) {
			m.Register(nil)
		}).
		HealthCheck(nil).
		Resource("test", func(r ConfigureResource) {
			r.Get(":id", nil)
			r.Update(":id", nil)
			r.Post(":id", nil)
			r.Delete(":id", nil)
			r.Resource("sub", func(subRes ConfigureResource) {
				r.Get(":id", nil)
			})
		}).
		Resource("test2", func(r ConfigureResource) {
			r.Get(":id", nil)
			r.Update(":id", nil)
			r.Post(":id", nil)
			r.Delete(":id", nil)
		})

	var handler QueryDataHandler
	return handler
}

func TransformProvider(configure ConfigurePlugin) TransformQueryDataHandler {
	configure.
		Metrics(func(m ConfigureMetricsCollector) {
			m.Register(nil)
		}).
		HealthCheck(nil).
		Resource("test", func(r ConfigureResource) {
			r.Get(":id", nil)
			r.Update(":id", nil)
			r.Post(":id", nil)
			r.Delete(":id", nil)
		}).
		Resource("test2", func(r ConfigureResource) {
			r.Get(":id", nil)
			r.Update(":id", nil)
			r.Post(":id", nil)
			r.Delete(":id", nil)
		})

	var handler TransformQueryDataHandler
	return handler
}
