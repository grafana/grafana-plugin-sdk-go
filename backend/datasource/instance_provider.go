package datasource

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tenant"
)

// InstanceFactoryFunc factory method for creating data source instances.
type InstanceFactoryFunc func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error)

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

func (ip *instanceProvider) GetKey(ctx context.Context, pluginContext backend.PluginContext) (interface{}, error) {
	if pluginContext.DataSourceInstanceSettings == nil {
		return nil, fmt.Errorf("data source instance settings cannot be nil")
	}

	defaultKey := pluginContext.DataSourceInstanceSettings.ID
	if tID := tenant.IDFromContext(ctx); tID != "" {
		return fmt.Sprintf("%s#%v", tID, defaultKey), nil
	}

	return defaultKey, nil
}

func (ip *instanceProvider) NeedsUpdate(_ context.Context, pluginContext backend.PluginContext, cachedInstance instancemgmt.CachedInstance) bool {
	curSettings := pluginContext.DataSourceInstanceSettings
	cachedSettings := cachedInstance.PluginContext.DataSourceInstanceSettings
	return !curSettings.Updated.Equal(cachedSettings.Updated)
}

func (ip *instanceProvider) NewInstance(_ context.Context, pluginContext backend.PluginContext) (instancemgmt.Instance, error) {
	return ip.factory(*pluginContext.DataSourceInstanceSettings)
}
