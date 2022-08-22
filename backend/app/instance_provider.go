package app

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// InstanceFactoryFunc factory method for creating data source instances.
type InstanceFactoryFunc func(settings backend.AppInstanceSettings) (instancemgmt.Instance, error)

// NewInstanceManager creates a new data source instance manager,
//
// This is a helper method for calling NewInstanceProvider and creating a new instancemgmt.InstanceProvider,
// and providing that to instancemgmt.New.
func NewInstanceManager(fn InstanceFactoryFunc) instancemgmt.InstanceManager {
	ip := NewInstanceProvider(fn)
	return instancemgmt.New(ip)
}

// NewInstanceProvider create a new data source instance provuder,
//
// The instance provider is responsible for providing cache keys for data source instances,
// creating new instances when needed and invalidating cached instances when they have been
// updated in Grafana.
// Cache key is based on the numerical data source identifier.
// If fn is nil, NewInstanceProvider panics.
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
	if pluginContext.AppInstanceSettings == nil {
		return nil, fmt.Errorf("app instance settings cannot be nil")
	}

	// Since app plugins have just one instance, use pluginID as instance cache key
	return pluginContext.PluginID, nil
}

func (ip *instanceProvider) NeedsUpdate(pluginContext backend.PluginContext, cachedInstance instancemgmt.CachedInstance) bool {
	curSettings := pluginContext.AppInstanceSettings
	cachedSettings := cachedInstance.PluginContext.AppInstanceSettings
	return !curSettings.Updated.Equal(cachedSettings.Updated)
}

func (ip *instanceProvider) NewInstance(pluginContext backend.PluginContext) (instancemgmt.Instance, error) {
	return ip.factory(*pluginContext.AppInstanceSettings)
}
