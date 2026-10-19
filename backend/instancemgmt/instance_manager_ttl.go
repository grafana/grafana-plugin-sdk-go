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
// Instances are evicted after the specified TTL. Disposal waits for requests
// using Acquire or Do to release the evicted instance.
func NewTTLInstanceManager(provider InstanceProvider) InstanceManager {
	return newTTLInstanceManager(provider, defaultInstanceTTL, defaultInstanceCleanupInterval)
}

func newTTLInstanceManager(provider InstanceProvider, instanceTTL, instanceCleanupInterval time.Duration) InstanceManager {
	if provider == nil {
		panic("provider cannot be nil")
	}

	cache := gocache.New(instanceTTL, instanceCleanupInterval)
	cache.OnEvicted(func(key string, value any) {
		ci := value.(CachedInstance)
		ci.managed.retire(0)
		backend.Logger.Debug("Evicted instance", "key", key)
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
	instance, _, err := im.get(ctx, pluginContext, false)
	return instance, err
}

func (im *instanceManagerWithTTL) Acquire(ctx context.Context, pluginContext backend.PluginContext) (Instance, func(), error) {
	return im.get(ctx, pluginContext, true)
}

func (im *instanceManagerWithTTL) get(ctx context.Context, pluginContext backend.PluginContext, tracked bool) (Instance, func(), error) {
	providerKey, err := im.provider.GetKey(ctx, pluginContext)
	if err != nil {
		return nil, nil, err
	}
	cacheKey, ok := providerKey.(string)
	if !ok {
		cacheKey = fmt.Sprintf("%v", providerKey)
	}

	// Serialize lookup and refresh with replacement. Otherwise a cache hit
	// could overwrite a newer instance with the stale value it just read.
	im.locker.Lock(cacheKey)
	defer im.locker.Unlock(cacheKey)

	if item, ok := im.cache.Get(cacheKey); ok {
		ci := item.(CachedInstance)
		if !im.provider.NeedsUpdate(ctx, pluginContext, ci) {
			// The janitor can evict independently of the key lock. Never acquire
			// an instance its callback has already retired.
			if release, ok := ci.managed.acquire(); ok {
				im.cache.SetDefault(cacheKey, ci)
				if !tracked {
					release()
					release = nil
				}
				return ci.instance, release, nil
			}
		}
	}

	instance, err := im.provider.NewInstance(ctx, pluginContext)
	if err != nil {
		// An update failure must not evict a still-live cached instance.
		return nil, nil, err
	}
	managed := &managedInstance{instance: instance}
	release, _ := managed.acquire()
	activeInstances.Inc()

	// Expired entries remain stored until a cleanup sweep. Delete explicitly
	// so replacing one also retires it instead of silently overwriting it.
	im.cache.Delete(cacheKey)
	im.cache.SetDefault(cacheKey, CachedInstance{
		PluginContext: pluginContext,
		instance:      instance,
		managed:       managed,
	})
	if !tracked {
		release()
		release = nil
	}
	return instance, release, nil
}

func (im *instanceManagerWithTTL) Do(ctx context.Context, pluginContext backend.PluginContext, fn InstanceCallbackFunc) error {
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
