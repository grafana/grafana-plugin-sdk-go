package instancemgmt

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestTTLInstanceManager(t *testing.T) {
	ctx := context.Background()
	pCtx := backend.PluginContext{
		OrgID: 1,
		AppInstanceSettings: &backend.AppInstanceSettings{
			Updated: time.Now(),
		},
	}

	tip := &testInstanceProvider{}
	im := NewTTLInstanceManager(tip)

	t.Run("When getting instance should create a new instance", func(t *testing.T) {
		instance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)
		require.Equal(t, pCtx.OrgID, instance.(*testInstance).orgID)
		require.Equal(t, pCtx.AppInstanceSettings.Updated, instance.(*testInstance).updated)

		t.Run("When getting instance should return same instance", func(t *testing.T) {
			instance2, err := im.Get(ctx, pCtx)
			require.NoError(t, err)
			require.Same(t, instance, instance2)
		})

		t.Run("When updating plugin context and getting instance", func(t *testing.T) {
			pCtxUpdated := backend.PluginContext{
				OrgID: 1,
				AppInstanceSettings: &backend.AppInstanceSettings{
					Updated: time.Now(),
				},
			}

			newInstance, err := im.Get(ctx, pCtxUpdated)

			t.Run("New instance should be created", func(t *testing.T) {
				require.NoError(t, err)
				require.NotNil(t, newInstance)
				require.Equal(t, pCtxUpdated.OrgID, newInstance.(*testInstance).orgID)
				require.Equal(t, pCtxUpdated.AppInstanceSettings.Updated, newInstance.(*testInstance).updated)
			})

			t.Run("New instance should not be the same as old instance", func(t *testing.T) {
				require.NotSame(t, instance, newInstance)
			})

			t.Run("Old instance should be disposed", func(t *testing.T) {
				instance.(*testInstance).wg.Wait()
				require.True(t, instance.(*testInstance).disposed.Load())
				require.Equal(t, int64(1), instance.(*testInstance).disposedTimes.Load())
			})
		})
	})
}

func TestTTLInstanceManagerWithCustomTTL(t *testing.T) {
	ctx := context.Background()
	pCtx := backend.PluginContext{
		OrgID: 1,
		AppInstanceSettings: &backend.AppInstanceSettings{
			Updated: time.Now(),
		},
	}

	tip := &testInstanceProvider{}
	ttl := 10 * time.Millisecond
	cleanupInterval := 5 * time.Millisecond
	im := newTTLInstanceManager(tip, ttl, cleanupInterval)

	t.Run("Instance should be evicted after TTL", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Tests with Sleep")
		}

		instance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)

		// Wait for TTL + cleanup interval + delta to ensure eviction + disposal
		time.Sleep(ttl + cleanupInterval + 10*time.Millisecond)

		// Get instance again - should create a new one since old one was evicted
		newInstance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, newInstance)
		require.NotSame(t, instance, newInstance)

		// Original instance should be disposed after cleanup interval
		instance.(*testInstance).wg.Wait()
		require.True(t, instance.(*testInstance).disposed.Load())
	})

	t.Run("Instance accessed before TTL expiry returns same instance", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Tests with Sleep")
		}

		instance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)

		// Access instance before TTL expires
		time.Sleep(ttl / 4) //  Wait 25% of TTL (well before expiry)
		sameInstance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.Same(t, instance, sameInstance)
		require.False(t, instance.(*testInstance).disposed.Load())
	})

	t.Run("Instance accessed before TTL expiry should reset TTL", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Tests with Sleep")
		}

		instance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)

		// Wait 7ms (70% of 10ms TTL) - close to expiry but not expired yet
		time.Sleep(7 * time.Millisecond)

		// Access instance before TTL expires - this should reset TTL
		sameInstance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.Same(t, instance, sameInstance)

		// Now wait 5ms more (total 12ms from instance creation)
		// Original TTL would have expired at 10ms, but reset TTL should be (7ms)+10ms = 17ms
		// So at 12ms, we should still have the same instance
		time.Sleep(5 * time.Millisecond)

		stillSameInstance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.Same(t, instance, stillSameInstance)
		require.False(t, instance.(*testInstance).disposed.Load())
	})
}

func TestTTLInstanceManagerConcurrency(t *testing.T) {
	t.Run("Check possible race condition issues when initially creating instance", func(t *testing.T) {
		ctx := context.Background()
		tip := &testInstanceProvider{}
		im := NewTTLInstanceManager(tip)
		pCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
		}
		var wg sync.WaitGroup
		wg.Add(10)

		var createdInstances []*testInstance
		mutex := new(sync.Mutex)
		// Creating new instances concurrently
		for i := 0; i < 10; i++ {
			go func() {
				instance, _ := im.Get(ctx, pCtx)
				mutex.Lock()
				defer mutex.Unlock()
				// Collect all instances created
				createdInstances = append(createdInstances, instance.(*testInstance))
				wg.Done()
			}()
		}
		wg.Wait()

		t.Run("All concurrent gets should return the same instance", func(t *testing.T) {
			// All instances should be the same (no race condition)
			firstInstance := createdInstances[0]
			for _, instance := range createdInstances {
				require.Same(t, firstInstance, instance)
			}
		})
	})

	t.Run("Check possible race condition issues when re-creating instance on settings update", func(t *testing.T) {
		ctx := context.Background()
		initialPCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
		}
		tip := &testInstanceProvider{}
		im := NewTTLInstanceManager(tip)
		// Creating initial instance with old contexts
		instanceToDispose, _ := im.Get(ctx, initialPCtx)

		updatedPCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
		}

		var wg sync.WaitGroup
		wg.Add(10)

		var createdInstances []*testInstance
		mutex := new(sync.Mutex)
		// Creating new instances because of updated context
		for i := 0; i < 10; i++ {
			go func() {
				instance, _ := im.Get(ctx, updatedPCtx)
				mutex.Lock()
				defer mutex.Unlock()
				// Collect all instances created during concurrent update
				createdInstances = append(createdInstances, instance.(*testInstance))
				wg.Done()
			}()
		}
		wg.Wait()

		t.Run("Initial instance should be disposed only once", func(t *testing.T) {
			instanceToDispose.(*testInstance).wg.Wait()
			require.Equal(t, int64(1), instanceToDispose.(*testInstance).disposedTimes.Load())
		})
		t.Run("All created instances should be the same (no race condition)", func(t *testing.T) {
			// All new instances should be the same
			if len(createdInstances) > 0 {
				firstInstance := createdInstances[0]
				for _, instance := range createdInstances {
					require.Same(t, firstInstance, instance)
				}
			}
		})
	})

	t.Run("Long recreation of instance should not affect other instances", func(t *testing.T) {
		const delay = time.Millisecond * 50
		ctx := context.Background()
		pCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
		}
		if testing.Short() {
			t.Skip("Tests with Sleep")
		}

		tip := &testInstanceProvider{delay: delay}
		im := NewTTLInstanceManager(tip)
		// Creating instance with id#1 in cache
		_, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		var wg1, wg2 sync.WaitGroup
		wg1.Add(1)
		wg2.Add(1)
		go func() {
			// Creating instance with id#2 in cache
			wg1.Done()
			_, err := im.Get(ctx, backend.PluginContext{
				OrgID: 2,
				AppInstanceSettings: &backend.AppInstanceSettings{
					Updated: time.Now(),
				},
			})
			require.NoError(t, err)
			wg2.Done()
		}()
		// Waiting before thread 2 starts to get the instance, so thread 2 could acquire the lock before thread 1
		wg1.Wait()
		// Getting existing instance with id#1 from cache
		start := time.Now()
		_, err = im.Get(ctx, pCtx)
		elapsed := time.Since(start)
		require.NoError(t, err)
		// Waiting before thread 2 finished to get the instance
		wg2.Wait()
		if elapsed > delay {
			require.Fail(t, "Instance should be retrieved from cache without delay")
		}
	})
}

func TestTTLInstanceManagerDo(t *testing.T) {
	ctx := context.Background()
	pCtx := backend.PluginContext{
		OrgID: 1,
		AppInstanceSettings: &backend.AppInstanceSettings{
			Updated: time.Now(),
		},
	}

	tip := &testInstanceProvider{}
	im := NewTTLInstanceManager(tip)

	t.Run("Do should execute callback with instance", func(t *testing.T) {
		var callbackInstance Instance
		err := im.Do(ctx, pCtx, func(instance Instance) {
			callbackInstance = instance
		})
		require.NoError(t, err)
		require.NotNil(t, callbackInstance)
		require.Equal(t, pCtx.OrgID, callbackInstance.(*testInstance).orgID)
	})

	t.Run("Do should panic with nil callback", func(t *testing.T) {
		require.Panics(t, func() {
			_ = im.Do(ctx, pCtx, nil)
		})
	})
}

func TestTTLInstanceManagerPanicHandling(t *testing.T) {
	t.Run("NewTTLInstanceManager should panic with nil provider", func(t *testing.T) {
		require.Panics(t, func() {
			NewTTLInstanceManager(nil)
		})
	})

	t.Run("newTTLInstanceManager should panic with nil provider", func(t *testing.T) {
		require.Panics(t, func() {
			newTTLInstanceManager(nil, defaultInstanceTTL, defaultInstanceCleanupInterval)
		})
	})
}
