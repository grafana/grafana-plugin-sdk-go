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
	return newTTLInstanceManager(provider, defaultInstanceTTL, defaultInstanceCleanupInterval, defaultDisposeTTL)
}

func newTTLInstanceManager(provider InstanceProvider, instanceTTL, instanceCleanupInterval, disposeTTL time.Duration) InstanceManager {
	if provider == nil {
		panic("provider cannot be nil")
	}

	cache := gocache.New(instanceTTL, instanceCleanupInterval)

	// Get evicts a live instance when it replaces it, so requests that already hold it get disposeTTL to finish.
	cache.OnEvicted(func(key string, value any) {
		ci := value.(CachedInstance)
		backend.Logger.Debug("Evicted instance", "key", key)
		time.AfterFunc(disposeTTL, func() {
			if disposer, valid := ci.instance.(InstanceDisposer); valid {
				disposer.Dispose()
			}
			activeInstances.Dec()
		})
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
	var cacheKey string
	if s, ok := providerKey.(string); ok {
		cacheKey = s
	} else {
		cacheKey = fmt.Sprintf("%v", providerKey)
	}
	if instance, ok := im.currentInstance(ctx, cacheKey, pluginContext); ok {
		return instance, nil
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
	}

	// Remove whatever is still stored under this key before replacing it. Get reports an
	// expired entry as a miss but leaves it in the cache until the next cleanup sweep, and
	// SetDefault would overwrite it without firing OnEvicted, so the old instance would
	// never be disposed and the active-instances gauge would never be decremented.
	im.cache.Delete(cacheKey)

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

// currentInstance holds the read lock so that refreshTTL cannot put back an instance that a concurrent replacement evicted.
func (im *instanceManagerWithTTL) currentInstance(ctx context.Context, cacheKey string, pluginContext backend.PluginContext) (Instance, bool) {
	im.locker.RLock(cacheKey)
	defer im.locker.RUnlock(cacheKey)

	item, ok := im.cache.Get(cacheKey)
	if !ok {
		return nil, false
	}
	ci := item.(CachedInstance)
	if im.provider.NeedsUpdate(ctx, pluginContext, ci) {
		return nil, false
	}
	im.refreshTTL(cacheKey, ci)
	return ci.instance, true
}

// refreshTTL updates the TTL of the cached instance by resetting its expiration time.
func (im *instanceManagerWithTTL) refreshTTL(cacheKey string, ci CachedInstance) {
	// SetDefault() technically creates a new cache entry with fresh TTL, effectively extending the instance's lifetime.
	im.cache.SetDefault(cacheKey, ci)
}
