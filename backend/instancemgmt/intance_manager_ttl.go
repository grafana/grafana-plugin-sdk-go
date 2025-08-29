package instancemgmt

import (
	"context"
	"fmt"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const (
	defaultInstanceTTL             = 1 * time.Hour
	defaultInstanceCleanupInterval = 2 * time.Hour
)

// NewTTLInstanceManager creates a new instance manager with TTL-based caching.
// Instances will be automatically evicted from the cache after the specified TTL.
func NewTTLInstanceManager(provider InstanceProvider) InstanceManager {
	return newTTLInstanceManager(provider, defaultInstanceTTL, defaultInstanceCleanupInterval)
}

func newTTLInstanceManager(provider InstanceProvider, instanceTTL, instanceCleanupInterval time.Duration) InstanceManager {
	if provider == nil {
		panic("provider cannot be nil")
	}

	// Use go-cache for TTL-based caching
	cache := gocache.New(instanceTTL, instanceCleanupInterval)

	// Set up the OnEvicted callback to dispose instances
	cache.OnEvicted(func(_ string, value interface{}) {
		ci := value.(CachedInstance)
		if disposer, valid := ci.instance.(InstanceDisposer); valid {
			disposer.Dispose()
		}
		activeInstances.Dec()
	})

	return &instanceManagerWithTTL{
		provider: provider,
		cache:    cache,
		locker:   newLocker(),
	}
}

type instanceManagerWithTTL struct {
	locker   *locker
	provider InstanceProvider
	cache    *gocache.Cache
}

func (im *instanceManagerWithTTL) Get(ctx context.Context, pluginContext backend.PluginContext) (Instance, error) {
	providerKey, err := im.provider.GetKey(ctx, pluginContext)
	if err != nil {
		return nil, err
	}
	// Double-checked locking for update/create criteria
	cacheKey := fmt.Sprintf("%v", providerKey)
	im.locker.RLock(cacheKey)
	item, ok := im.cache.Get(cacheKey)
	im.locker.RUnlock(cacheKey)
	if ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
			im.locker.Lock(cacheKey)
			im.refreshTTL(cacheKey, ci)
			im.locker.Unlock(cacheKey)
			return ci.instance, nil
		}
	}

	im.locker.Lock(cacheKey)
	defer im.locker.Unlock(cacheKey)

	if item, ok := im.cache.Get(cacheKey); ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
			im.refreshTTL(cacheKey, ci)
			return ci.instance, nil
		}

		im.cache.Delete(cacheKey)
	}

	instance, err := im.provider.NewInstance(ctx, pluginContext)
	if err != nil {
		return nil, err
	}
	im.cache.SetDefault(cacheKey, CachedInstance{
		PluginContext: pluginContext,
		instance:      instance,
	})
	activeInstances.Inc()

	return instance, nil
}

func (im *instanceManagerWithTTL) Do(ctx context.Context, pluginContext backend.PluginContext, fn InstanceCallbackFunc) error {
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

// refreshTTL updates the TTL of the cached instance by resetting its expiration time.
func (im *instanceManagerWithTTL) refreshTTL(cacheKey string, ci CachedInstance) {
	// SetDefault() technically creates a new cache entry with fresh TTL, effectively extending the instance's lifetime.
	im.cache.SetDefault(cacheKey, ci)
}
