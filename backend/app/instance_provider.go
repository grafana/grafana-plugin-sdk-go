package app

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// InstanceFactoryFunc factory method for creating app instances.
type InstanceFactoryFunc func(settings backend.AppInstanceSettings) (instancemgmt.Instance, error)

// NewInstanceProvider create a new app instance provuder,
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

	return pluginContext.OrgID, nil
}

func (ip *instanceProvider) NeedsUpdate(pluginContext backend.PluginContext, cachedInstance instancemgmt.CachedInstance) bool {
	curSettings := pluginContext.AppInstanceSettings
	cachedSettings := cachedInstance.PluginContext.AppInstanceSettings
	return !curSettings.Updated.Equal(cachedSettings.Updated)
}

func (ip *instanceProvider) NewInstance(pluginContext backend.PluginContext) (instancemgmt.Instance, error) {
	return ip.factory(*pluginContext.AppInstanceSettings)
}
