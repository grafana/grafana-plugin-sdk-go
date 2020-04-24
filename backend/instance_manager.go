package backend

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
)

// InstanceDisposer interface marking
type InstanceDisposer interface {
	Dispose()
}

// AppInstance defines the interface for application plugin instances.
// There will be a maximum of one instance per Grafana organization.
type AppInstance interface {
	CheckHealthHandler
	CallResourceHandler
}

// AppInstanceProviderFunc is the factory method for creating a new application plugin instance.
type AppInstanceProviderFunc func(settings AppInstanceSettings) (AppInstance, error)

// DataSourceInstance defines the interface for data source plugin instances.
type DataSourceInstance interface {
	CheckHealthHandler
	CallResourceHandler
	QueryDataHandler
}

// DataSourceInstanceProviderFunc is the factory method for creating a new data source plugin instance.
type DataSourceInstanceProviderFunc func(settings DataSourceInstanceSettings) (DataSourceInstance, error)

// InstanceProviders providers for creating plugin instances.
type InstanceProviders struct {
	// AppInstanceProvider provider for creating an app instance.
	AppInstanceProvider AppInstanceProviderFunc

	// DataSourceInstanceProvider provider for creating a data source instance.
	DataSourceInstanceProvider DataSourceInstanceProviderFunc
}

// InstanceManager manages the lifecycle of plugin instances.
type InstanceManager interface {
	// Serve starts serving the plugin over gRPC.
	Serve() error
}

type appInstanceInfo struct {
	// The raw GRPC values that create the instance
	settings AppInstanceSettings

	// The specific instance
	instance AppInstance
}

type dataSourceInstanceInfo struct {
	// The raw GRPC values that create the instance
	settings DataSourceInstanceSettings

	// The specific instance
	instance DataSourceInstance
}

type instanceManager struct {
	rwMutex          sync.RWMutex
	providers        InstanceProviders
	appInstanceCache map[int64]appInstanceInfo
	dsInstanceCache  map[int64]dataSourceInstanceInfo
}

// NewInstanceManager creates a new instance manager.
func NewInstanceManager(providers InstanceProviders) InstanceManager {
	if providers.AppInstanceProvider == nil {
		providers.AppInstanceProvider = newFallbackAppInstance
	}
	return &instanceManager{providers: providers}
}

func (im *instanceManager) Serve() error {
	return Serve(ServeOpts{
		CheckHealthHandler:  im,
		CallResourceHandler: im,
		QueryDataHandler:    im,
	})
}

func (im *instanceManager) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	if req.PluginContext.DataSourceInstanceSettings != nil {
		dsInstance, err := im.getDataSourceInstance(req.PluginContext)
		if err != nil {
			return nil, err
		}
		return dsInstance.CheckHealth(ctx, req)
	}

	appInstance, err := im.getAppInstance(req.PluginContext)
	if err != nil {
		return nil, err
	}

	return appInstance.CheckHealth(ctx, req)
}

func (im *instanceManager) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	if req.PluginContext.DataSourceInstanceSettings != nil {
		dsInstance, err := im.getDataSourceInstance(req.PluginContext)
		if err != nil {
			return err
		}

		return dsInstance.CallResource(ctx, req, sender)
	}

	appInstance, err := im.getAppInstance(req.PluginContext)
	if err != nil {
		return err
	}

	return appInstance.CallResource(ctx, req, sender)
}

func (im *instanceManager) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	if req.PluginContext.DataSourceInstanceSettings != nil {
		dsInstance, err := im.getDataSourceInstance(req.PluginContext)
		if err != nil {
			return nil, err
		}

		return dsInstance.QueryData(ctx, req)
	}

	return nil, fmt.Errorf("only data source supports QueryData")
}

func (im *instanceManager) getAppInstance(pluginContext PluginContext) (AppInstance, error) {
	if pluginContext.AppInstanceSettings == nil {
		return nil, fmt.Errorf("app instance settings required")
	}

	settings := *pluginContext.AppInstanceSettings
	cacheKey := pluginContext.OrgID

	// Aquire read lock
	im.rwMutex.RLock()
	info, hasCachedInstance := im.appInstanceCache[cacheKey]
	// Release read lock
	im.rwMutex.RUnlock()

	// return fast if cached instance havent't been updated
	if hasCachedInstance && settings.Updated.Equal(info.settings.Updated) {
		return info.instance, nil
	}

	// Aquire write lock
	im.rwMutex.Lock()
	defer im.rwMutex.Unlock()

	if hasCachedInstance {
		// disposed instance implementing the InstanceDisposer interface
		if disposer, isDisposer := info.instance.(InstanceDisposer); isDisposer {
			disposer.Dispose()
		}
	}

	// Create a new instance
	instance, err := im.providers.AppInstanceProvider(settings)
	if err != nil {
		return nil, err
	}

	info = appInstanceInfo{
		settings: settings,
		instance: instance,
	}

	// Set the instance for the key (will replace the old value if exists)
	im.appInstanceCache[cacheKey] = info
	return info.instance, nil
}

func (im *instanceManager) getDataSourceInstance(pluginContext PluginContext) (DataSourceInstance, error) {
	if im.providers.DataSourceInstanceProvider == nil {
		return nil, fmt.Errorf("no data source instance provider")
	}

	if pluginContext.DataSourceInstanceSettings == nil {
		return nil, fmt.Errorf("data source instance settings required")
	}

	settings := *pluginContext.DataSourceInstanceSettings
	cacheKey := settings.ID

	// Aquire read lock
	im.rwMutex.RLock()
	info, hasCachedInstance := im.dsInstanceCache[cacheKey]
	// Release read lock
	im.rwMutex.RUnlock()

	// return fast if cached instance havent't been updated
	if hasCachedInstance && settings.Updated.Equal(info.settings.Updated) {
		return info.instance, nil
	}

	// Aquire write lock
	im.rwMutex.Lock()
	defer im.rwMutex.Unlock()

	if hasCachedInstance {
		// disposed instance implementing the InstanceDisposer interface
		if disposer, isDisposer := info.instance.(InstanceDisposer); isDisposer {
			disposer.Dispose()
		}
	}

	// Create a new instance
	instance, err := im.providers.DataSourceInstanceProvider(settings)
	if err != nil {
		return nil, err
	}

	info = dataSourceInstanceInfo{
		settings: settings,
		instance: instance,
	}

	// Set the instance for the key (will replace the old value if exists)
	im.dsInstanceCache[cacheKey] = info
	return info.instance, nil
}

type fallbackAppInstance struct {
}

func newFallbackAppInstance(settings AppInstanceSettings) (AppInstance, error) {
	return &fallbackAppInstance{}, nil
}

func (app *fallbackAppInstance) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	return &CheckHealthResult{
		Status:  HealthStatusOk,
		Message: "Plugin is running",
	}, nil
}

func (app *fallbackAppInstance) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	return sender.Send(&CallResourceResponse{
		Status: http.StatusNotImplemented,
	})
}

type myAppInstance struct {
}

func newAppInstance(settings AppInstanceSettings) (AppInstance, error) {
	return &myAppInstance{}, nil
}

func (app *myAppInstance) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	return nil, nil
}

func (app *myAppInstance) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	return nil
}

type myDataSourceInstance struct {
}

func newDataSourceInstance(settings DataSourceInstanceSettings) (DataSourceInstance, error) {
	return &myDataSourceInstance{}, nil
}

func (ds *myDataSourceInstance) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	return nil, nil
}

func (ds *myDataSourceInstance) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	return nil
}

func (ds *myDataSourceInstance) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	return nil, nil
}

func MainInstanceExample() {
	im := NewInstanceManager(InstanceProviders{
		AppInstanceProvider:        newAppInstance,
		DataSourceInstanceProvider: newDataSourceInstance,
	})
	err := im.Serve()
	if err != nil {
		Logger.Error(err.Error())
		os.Exit(1)
	}
}
