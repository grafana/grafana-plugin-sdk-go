package instancemgmt

import (
	"context"
	"reflect"
	"sync"
	"time"

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

const defaultDisposeTTL = 5 * time.Second // Time to wait before disposing an instance

// Instance is a marker interface for an instance.
type Instance any

// InstanceDisposer is implemented by an Instance that has a Dispose method,
// which defines that the instance is disposable.
//
// InstanceManager calls Dispose after an instance is replaced and its disposal
// delay has elapsed. Requests using Acquire or Do also keep the instance alive
// until they finish.
type InstanceDisposer interface {
	Dispose()
}

// InstanceCallbackFunc defines the callback function of the InstanceManager.Do method.
// The argument provided will of type Instance.
type InstanceCallbackFunc any

// InstanceManager manages the lifecycle of instances.
type InstanceManager interface {
	// Get returns an Instance. Its lifetime is not tracked after Get returns;
	// use Acquire or Do when the instance must remain alive during a request.
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
	managed       *managedInstance
}

// InstanceProvider defines an instance provider, providing instances.
type InstanceProvider interface {
	// GetKey returns a cache key to be used for caching an Instance.
	GetKey(ctx context.Context, pluginContext backend.PluginContext) (any, error)

	// NeedsUpdate returns whether a cached Instance have been updated.
	NeedsUpdate(ctx context.Context, pluginContext backend.PluginContext, cachedInstance CachedInstance) bool

	// NewInstance creates a new Instance.
	NewInstance(ctx context.Context, pluginContext backend.PluginContext) (Instance, error)
}

// New create a new instance manager.
func New(provider InstanceProvider) InstanceManager {
	return newInstanceManager(provider, defaultDisposeTTL)
}

func newInstanceManager(provider InstanceProvider, disposeTTL time.Duration) *instanceManager {
	if provider == nil {
		panic("provider cannot be nil")
	}

	if disposeTTL <= 0 {
		disposeTTL = defaultDisposeTTL
	}

	return &instanceManager{
		provider:   provider,
		cache:      sync.Map{},
		locker:     newLocker(),
		disposeTTL: disposeTTL,
	}
}

type instanceManager struct {
	locker     *locker
	provider   InstanceProvider
	cache      sync.Map
	disposeTTL time.Duration
}

func (im *instanceManager) Get(ctx context.Context, pluginContext backend.PluginContext) (Instance, error) {
	instance, _, err := im.get(ctx, pluginContext, false)
	return instance, err
}

func (im *instanceManager) Acquire(ctx context.Context, pluginContext backend.PluginContext) (Instance, func(), error) {
	return im.get(ctx, pluginContext, true)
}

func (im *instanceManager) get(ctx context.Context, pluginContext backend.PluginContext, tracked bool) (Instance, func(), error) {
	cacheKey, err := im.provider.GetKey(ctx, pluginContext)
	if err != nil {
		return nil, nil, err
	}
	// Pin the instance before releasing the key lock, so a concurrent swap
	// cannot retire it between lookup and acquisition.
	im.locker.RLock(cacheKey)
	if item, ok := im.cache.Load(cacheKey); ok {
		ci := item.(CachedInstance)
		if !im.provider.NeedsUpdate(ctx, pluginContext, ci) {
			var release func()
			if tracked {
				release, _ = ci.managed.acquire()
			}
			im.locker.RUnlock(cacheKey)
			return ci.instance, release, nil
		}
	}
	im.locker.RUnlock(cacheKey)

	im.locker.Lock(cacheKey)
	defer im.locker.Unlock(cacheKey)

	var previous *managedInstance
	if item, ok := im.cache.Load(cacheKey); ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
			var release func()
			if tracked {
				release, _ = ci.managed.acquire()
			}
			return ci.instance, release, nil
		}
		previous = ci.managed
	}

	instance, err := im.provider.NewInstance(ctx, pluginContext)
	if err != nil {
		// Keep the previous instance cached and undisposed so a failed update doesn't strand it.
		return nil, nil, err
	}
	managed := &managedInstance{instance: instance}
	var release func()
	if tracked {
		release, _ = managed.acquire()
	}
	im.cache.Store(cacheKey, CachedInstance{
		PluginContext: pluginContext,
		instance:      instance,
		managed:       managed,
	})
	activeInstances.Inc()

	if previous != nil {
		previous.retire(im.disposeTTL)
	}

	return instance, release, nil
}

func (im *instanceManager) Do(ctx context.Context, pluginContext backend.PluginContext, fn InstanceCallbackFunc) error {
	if fn == nil {
		panic("fn cannot be nil")
	}

	instance, release, err := im.Acquire(ctx, pluginContext)
	if err != nil {
		return err
	}
	defer release()

	callInstanceHandlerFunc(fn, instance)
	return nil
}

func callInstanceHandlerFunc(fn InstanceCallbackFunc, instance any) {
	var params = []reflect.Value{}
	params = append(params, reflect.ValueOf(instance))
	reflect.ValueOf(fn).Call(params)
}
