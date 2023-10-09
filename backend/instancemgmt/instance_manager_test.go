package instancemgmt

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestInstanceManager(t *testing.T) {
	ctx := context.Background()
	pCtx := backend.PluginContext{
		OrgID: 1,
		AppInstanceSettings: &backend.AppInstanceSettings{
			Updated: time.Now(),
		},
	}

	tip := &testInstanceProvider{}
	im := New(tip)

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

			t.Run("Old instance should only be disposed after subsequent call to get instance", func(t *testing.T) {
				require.False(t, instance.(*testInstance).disposed)

				_, err = im.Get(ctx, pCtxUpdated)
				require.NoError(t, err)

				require.True(t, instance.(*testInstance).disposed)
				require.Equal(t, int64(1), instance.(*testInstance).disposedTimes, "Instance should be disposed only once")
			})
		})
	})
}

func TestInstanceManagerConcurrency(t *testing.T) {
	t.Run("Check possible race condition issues when initially creating instance", func(t *testing.T) {
		ctx := context.Background()
		tip := &testInstanceProvider{}
		im := New(tip)
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
		// Creating new instances because of updated context
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

		t.Run("All created instances should be either disposed or exist in cache for later disposing", func(t *testing.T) {
			cachedInstance, _ := im.Get(ctx, pCtx)
			for _, instance := range createdInstances {
				if cachedInstance.(*testInstance) != instance && instance.disposedTimes < 1 {
					require.FailNow(t, "Found lost reference to un-disposed instance")
				}
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
		im := New(tip)
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
			require.True(t, instanceToDispose.(*testInstance).disposed)
			require.Equal(t, int64(1), instanceToDispose.(*testInstance).disposedTimes, "Instance should be disposed only once")
		})
		t.Run("All created instances should be either disposed or exist in cache for later disposing", func(t *testing.T) {
			cachedInstance, _ := im.Get(ctx, updatedPCtx)
			for _, instance := range createdInstances {
				if cachedInstance.(*testInstance) != instance && instance.disposedTimes < 1 {
					require.FailNow(t, "Found lost reference to un-disposed instance")
				}
			}
		})
	})

	t.Run("Long recreation of instance should not affect datasources with different ID", func(t *testing.T) {
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
		im := New(tip)
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
		// Waiting before thread 2 starts to get the instance, so thread 2 could qcquire the lock before thread 1
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

type testInstance struct {
	orgID         int64
	updated       time.Time
	disposed      bool
	disposedTimes int64
}

func (ti *testInstance) Dispose() {
	ti.disposed = true
	atomic.AddInt64(&ti.disposedTimes, 1)
}

type testInstanceProvider struct {
	getKeyFunc      func(ctx context.Context, pluginContext backend.PluginContext) (interface{}, error)
	needsUpdateFunc func(ctx context.Context, pluginContext backend.PluginContext, cachedInstance CachedInstance) bool
	newInstanceFunc func(ctx context.Context, pluginContext backend.PluginContext) (Instance, error)

	delay time.Duration
}

func (tip *testInstanceProvider) GetKey(_ context.Context, pluginContext backend.PluginContext) (interface{}, error) {
	if tip.getKeyFunc != nil {
		return tip.getKeyFunc(context.Background(), pluginContext)
	}
	return pluginContext.OrgID, nil
}

func (tip *testInstanceProvider) NeedsUpdate(_ context.Context, pluginContext backend.PluginContext, cachedInstance CachedInstance) bool {
	if tip.needsUpdateFunc != nil {
		return tip.needsUpdateFunc(context.Background(), pluginContext, cachedInstance)
	}
	curUpdated := pluginContext.AppInstanceSettings.Updated
	cachedUpdated := cachedInstance.PluginContext.AppInstanceSettings.Updated
	return !curUpdated.Equal(cachedUpdated)
}

func (tip *testInstanceProvider) NewInstance(_ context.Context, pluginContext backend.PluginContext) (Instance, error) {
	if tip.newInstanceFunc != nil {
		return tip.newInstanceFunc(context.Background(), pluginContext)
	}
	if tip.delay > 0 {
		time.Sleep(tip.delay)
	}
	return &testInstance{
		orgID:   pluginContext.OrgID,
		updated: pluginContext.AppInstanceSettings.Updated,
	}, nil
}

func TestInstanceManager_DisposableInstances(t *testing.T) {
	di1Disposed := 0
	di1 := newDisposableInstance(func() {
		di1Disposed++
	})

	ip := &testInstanceProvider{
		getKeyFunc: func(ctx context.Context, pluginContext backend.PluginContext) (interface{}, error) {
			return "123", nil
		},
		newInstanceFunc: func(ctx context.Context, pluginContext backend.PluginContext) (Instance, error) {
			return di1, nil
		},
	}

	// Create instance manager and get instance saved into cache
	im := New(ip)
	i, err := im.Get(context.Background(), backend.PluginContext{})
	require.NoError(t, err)
	require.Equal(t, di1, i)
	require.Equal(t, 0, di1Disposed)

	err = i.(*disposableInstance).DoWork()
	require.NoError(t, err)

	// update instance provider mock to ensure the needsUpdateFunc to always return true so that next call to im.Get
	// will be forced to call newInstanceFunc for the cache entry with key "123"
	ip.needsUpdateFunc = func(ctx context.Context, pluginContext backend.PluginContext, cachedInstance CachedInstance) bool {
		return true
	}
	ip.getKeyFunc = func(ctx context.Context, pluginContext backend.PluginContext) (interface{}, error) {
		return "123", nil
	}
	ip.newInstanceFunc = func(ctx context.Context, pluginContext backend.PluginContext) (Instance, error) {
		return newDisposableInstance(func() {}), nil
	}

	i, err = im.Get(context.Background(), backend.PluginContext{})
	require.NoError(t, err)
	require.NotSame(t, di1, i)
	require.Equal(t, 0, di1Disposed)

	// di1 instance is still valid and not disposed
	err = di1.DoWork()
	require.NoError(t, err)

	// di1 instance is disposed after subsequent call to im.Get
	_, err = im.Get(context.Background(), backend.PluginContext{})
	require.NoError(t, err)

	err = di1.DoWork()
	require.Error(t, err)
}

type disposableInstance struct {
	m           sync.RWMutex
	disposeFunc func()
	disposed    bool
}

func newDisposableInstance(f func()) *disposableInstance {
	return &disposableInstance{
		disposeFunc: f,
	}
}

func (di *disposableInstance) DoWork() error {
	di.m.RLock()
	defer di.m.RUnlock()
	if di.disposed {
		return errors.New("OH NO")
	}
	return nil
}

func (di *disposableInstance) Dispose() {
	di.m.Lock()
	di.disposed = true
	di.m.Unlock()
	if di.disposeFunc != nil {
		di.disposeFunc()
	}
}
