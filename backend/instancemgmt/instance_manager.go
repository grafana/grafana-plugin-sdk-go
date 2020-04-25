package instancemgmt

import (
	"reflect"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// InstanceDisposer is implemented by any instance that has a Dispose method,
// which defines that the instance is disposable.
type InstanceDisposer interface {
	Dispose()
}

// Instance is a marker interface for an instance.
type Instance interface{}

// InstanceCallbackFunc defines the callback function of the InstanceManager.Do method.
// The argument provided will of type Instance.
type InstanceCallbackFunc interface{}

// InstanceManager manages the lifecycle of instances.
type InstanceManager interface {
	Get(pluginContext backend.PluginContext) (Instance, error)
	Do(pluginContext backend.PluginContext, fn InstanceCallbackFunc) error
}

// CachedInstance a cached instance.
type CachedInstance struct {
	PluginContext backend.PluginContext
	instance      Instance
}

// InstanceProvider defines an instance provider, providing instances.
type InstanceProvider interface {
	GetKey(pluginContext backend.PluginContext) (interface{}, error)
	NeedsUpdate(pluginContext backend.PluginContext, cachedInstance CachedInstance) bool
	NewInstance(pluginContext backend.PluginContext) (Instance, error)
}

// New create a new instance manager.
func New(provider InstanceProvider) InstanceManager {
	if provider == nil {
		panic("provider cannot be nil")
	}

	return &instanceManager{
		provider: provider,
		cache:    map[interface{}]CachedInstance{},
	}
}

type instanceManager struct {
	rwMutex  sync.RWMutex
	provider InstanceProvider
	cache    map[interface{}]CachedInstance
}

func (im *instanceManager) Get(pluginContext backend.PluginContext) (Instance, error) {
	cacheKey, err := im.provider.GetKey(pluginContext)
	if err != nil {
		return nil, err
	}
	im.rwMutex.RLock()
	ci, ok := im.cache[cacheKey]
	im.rwMutex.RUnlock()

	if ok {
		needsUpdate := im.provider.NeedsUpdate(pluginContext, ci)

		if !needsUpdate {
			return ci.instance, nil
		}

		if disposer, valid := ci.instance.(InstanceDisposer); valid {
			disposer.Dispose()
		}
	}

	im.rwMutex.Lock()
	defer im.rwMutex.Unlock()

	instance, err := im.provider.NewInstance(pluginContext)
	if err != nil {
		return nil, err
	}
	im.cache[cacheKey] = CachedInstance{
		PluginContext: pluginContext,
		instance:      instance,
	}

	return instance, nil
}

func (im *instanceManager) Do(pluginContext backend.PluginContext, fn InstanceCallbackFunc) error {
	if fn == nil {
		panic("fn cannot be nil")
	}

	instance, err := im.Get(pluginContext)
	if err != nil {
		return err
	}

	callInstanceHandlerFunc(fn, instance)
	return nil
}

func callInstanceHandlerFunc(fn InstanceCallbackFunc, instance interface{}) {
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(instance))
	reflect.ValueOf(fn).Call(params)
}
