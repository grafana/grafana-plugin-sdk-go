package datasource

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// InstanceFactoryFunc factory method for creating data source instances.
type InstanceFactoryFunc func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error)

// NewInstanceProvider create a new data source instance provuder,
//
// The instance provider is responsible for providing cache keys for data source instances,
// creating new instances when needed and invalidating cached instances when they have been
// updated in Grafana.
// Cache key is based on the numerical data source identifier.
func NewInstanceProvider(fn InstanceFactoryFunc) instancemgmt.InstanceProvider {
	if fn == nil {
		panic("fn cannot be nil")
	}

	return &instanceProvider{
		factory: fn,
	}
}

type instanceProvider struct {
	factory InstanceFactoryFunc
}

func (ip *instanceProvider) GetKey(pluginContext backend.PluginContext) (interface{}, error) {
	if pluginContext.DataSourceInstanceSettings == nil {
		return nil, fmt.Errorf("data source instance settings cannot be nil")
	}

	return pluginContext.DataSourceInstanceSettings.ID, nil
}

func (ip *instanceProvider) NeedsUpdate(pluginContext backend.PluginContext, cachedInstance instancemgmt.CachedInstance) bool {
	curSettings := pluginContext.DataSourceInstanceSettings
	cachedSettings := cachedInstance.PluginContext.DataSourceInstanceSettings
	return !curSettings.Updated.Equal(cachedSettings.Updated)
}

func (ip *instanceProvider) NewInstance(pluginContext backend.PluginContext) (instancemgmt.Instance, error) {
	return ip.factory(*pluginContext.DataSourceInstanceSettings)
}
