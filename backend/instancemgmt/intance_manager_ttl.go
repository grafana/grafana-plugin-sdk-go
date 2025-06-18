package instancemgmt

import (
	"context"
	"fmt"

	gocache "github.com/patrickmn/go-cache"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

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
	item, ok := im.cache.Get(cacheKey)

	if ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
			return ci.instance, nil
		}
	}

	im.locker.Lock(cacheKey)
	defer im.locker.Unlock(cacheKey)

	if item, ok := im.cache.Get(cacheKey); ok {
		ci := item.(CachedInstance)
		needsUpdate := im.provider.NeedsUpdate(ctx, pluginContext, ci)

		if !needsUpdate {
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
