package instancemgmt

import (
	"context"
	"reflect"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

var (
	activeInstances = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "plugins",
		Name:      "active_instances",
		Help:      "The number of active plugin instances",
	})
)

// Instance is a marker interface for an instance.
type Instance interface{}

// InstanceDisposer is implemented by an Instance that has a Dispose method,
// which defines that the instance is disposable.
//
// InstanceManager will call the Dispose method before an Instance is replaced
// with a new Instance. This allows an Instance to clean up resources in use,
// if any.
type InstanceDisposer interface {
	Dispose()
}

// InstanceCallbackFunc defines the callback function of the InstanceManager.Do method.
// The argument provided will of type Instance.
type InstanceCallbackFunc interface{}

// InstanceManager manages the lifecycle of instances.
type InstanceManager interface {
	// Get returns an Instance.
	//
	// If Instance is cached and not updated it's returned. If Instance is not cached or
	// updated, a new Instance is created and cached before returned.
	Get(ctx context.Context, pluginContext backend.PluginContext) (Instance, error)

	// Do provides an Instance as argument to fn.
	//
	// If Instance is cached and not updated provides as argument to fn. If Instance is not cached or
	// updated, a new Instance is created and cached before provided as argument to fn.
	Do(ctx context.Context, pluginContext backend.PluginContext, fn InstanceCallbackFunc) error
}

// CachedInstance a cached Instance.
type CachedInstance struct {
	PluginContext backend.PluginContext
	instance      Instance
}

// InstanceProvider defines an instance provider, providing instances.
type InstanceProvider interface {
	// GetKey returns a cache key to be used for caching an Instance.
	GetKey(ctx context.Context, pluginContext backend.PluginContext) (interface{}, error)

	// NeedsUpdate returns whether a cached Instance have been updated.
	NeedsUpdate(ctx context.Context, pluginContext backend.PluginContext, cachedInstance CachedInstance) bool

	// NewInstance creates a new Instance.
	NewInstance(ctx context.Context, pluginContext backend.PluginContext) (Instance, error)
}

// New create a new instance manager.
func New(provider InstanceProvider) InstanceManager {
	if provider == nil {
		panic("provider cannot be nil")
	}

	return &instanceManager{
		provider:         provider,
		cache:            sync.Map{},
		locker:           newLocker(),
		instanceDisposer: newInstanceDisposer(),
	}
}

type instanceManager struct {
	locker           *locker
	provider         InstanceProvider
	cache            sync.Map
	instanceDisposer instanceDisposer
}

func (im *instanceManager) Get(ctx context.Context, pluginContext backend.PluginContext) (Instance, error) {
	cacheKey, err := im.provider.GetKey(ctx, pluginContext)
	if err != nil {
		return nil, err
	}
	// Double-checked locking for update/create criteria
	im.locker.RLock(cacheKey)
	item, ok := im.cache.Load(cacheKey)
	im.locker.RUnlock(cacheKey)

	if ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
			if im.instanceDisposer.tracking(cacheKey) {
				im.instanceDisposer.dispose(cacheKey)
			}
			return ci.instance, nil
		}
	}

	im.locker.Lock(cacheKey)
	defer im.locker.Unlock(cacheKey)

	if item, ok := im.cache.Load(cacheKey); ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
			if im.instanceDisposer.tracking(cacheKey) {
				im.instanceDisposer.dispose(cacheKey)
			}
			return ci.instance, nil
		}

		if id, ok := im.instanceDisposer.disposable(ci.instance); ok {
			im.instanceDisposer.track(cacheKey, id)
		}
	}

	instance, err := im.provider.NewInstance(ctx, pluginContext)
	if err != nil {
		return nil, err
	}
	im.cache.Store(cacheKey, CachedInstance{
		PluginContext: pluginContext,
		instance:      instance,
	})
	activeInstances.Inc()

	return instance, nil
}

func (im *instanceManager) Do(ctx context.Context, pluginContext backend.PluginContext, fn InstanceCallbackFunc) error {
	if fn == nil {
		panic("fn cannot be nil")
	}

	instance, err := im.Get(ctx, pluginContext)
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

// instanceDisposer tracks and disposes of disposable instances.
type instanceDisposer struct {
	disposableCache sync.Map
	m               sync.RWMutex
}

func newInstanceDisposer() instanceDisposer {
	return instanceDisposer{
		disposableCache: sync.Map{},
	}
}

// disposable returns whether an instance is disposable.
func (id *instanceDisposer) disposable(i Instance) (InstanceDisposer, bool) {
	if d, ok := i.(InstanceDisposer); ok {
		return d, true
	}
	return nil, false
}

// track tracks a disposable instance. If there is a disposable instance already, we should dispose of it first.
func (id *instanceDisposer) track(cacheKey interface{}, disposableInstance InstanceDisposer) {
	id.m.Lock()
	defer id.m.Unlock()

	// If there is a disposable instance already, we should dispose of it first.
	if _, exists := id.disposableCache.Load(cacheKey); exists {
		id.disposableCache.Delete(cacheKey)
	}

	id.disposableCache.Store(cacheKey, disposableInstance)
}

// disposeIfExists disposes of a disposable instance.
func (id *instanceDisposer) tracking(cacheKey interface{}) bool {
	id.m.RLock()
	defer id.m.RUnlock()
	_, exists := id.disposableCache.Load(cacheKey)
	return exists
}

// disposeIfExists disposes of a disposable instance.
func (id *instanceDisposer) dispose(cacheKey interface{}) {
	id.m.Lock()
	defer id.m.Unlock()
	id.disposableCache.Delete(cacheKey)
}
