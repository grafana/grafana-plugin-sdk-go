package instancemgmt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestAcquiredInstanceSurvivesReplacement(t *testing.T) {
	for _, ttl := range []bool{false, true} {
		name := "standard"
		if ttl {
			name = "TTL"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			provider := &testInstanceProvider{}
			var manager InstanceManager = newInstanceManager(provider, time.Millisecond)
			if ttl {
				manager = newTTLInstanceManager(provider, time.Hour, time.Hour)
			}
			pc := backend.PluginContext{AppInstanceSettings: &backend.AppInstanceSettings{Updated: time.Now()}}
			first, releaseA, err := Acquire(ctx, manager, pc)
			require.NoError(t, err)
			defer releaseA()
			second, releaseB, err := Acquire(ctx, manager, pc)
			require.NoError(t, err)
			defer releaseB()
			require.True(t, first == second)

			pc.AppInstanceSettings = &backend.AppInstanceSettings{Updated: pc.AppInstanceSettings.Updated.Add(time.Second)}
			replacement, err := manager.Get(ctx, pc)
			require.NoError(t, err)
			require.True(t, first != replacement)
			old := first.(*testInstance)
			require.Never(t, old.disposed.Load, 20*time.Millisecond, time.Millisecond)
			releaseA()
			releaseA() // Release is safe to call more than once.
			require.False(t, old.disposed.Load())
			releaseB()
			require.Eventually(t, func() bool { return old.disposedTimes.Load() == 1 }, time.Second, time.Millisecond)
		})
	}
}

func TestTTLFailedUpdatePreservesInstance(t *testing.T) {
	ctx := context.Background()
	provider := &testInstanceProvider{}
	manager := newTTLInstanceManager(provider, time.Hour, time.Hour)
	pc := backend.PluginContext{AppInstanceSettings: &backend.AppInstanceSettings{Updated: time.Now()}}
	first, err := manager.Get(ctx, pc)
	require.NoError(t, err)
	updated := pc
	updated.AppInstanceSettings = &backend.AppInstanceSettings{Updated: pc.AppInstanceSettings.Updated.Add(time.Second)}
	provider.err = errors.New("update failed")
	for range 3 {
		_, err := manager.Get(ctx, updated)
		require.ErrorIs(t, err, provider.err)
	}
	cached, err := manager.Get(ctx, pc)
	require.NoError(t, err)
	require.True(t, first == cached)
	require.False(t, first.(*testInstance).disposed.Load())
	provider.err = nil
	_, err = manager.Get(ctx, updated)
	require.NoError(t, err)
	require.Equal(t, int64(1), first.(*testInstance).disposedTimes.Load())
}

func TestTTLAcquiredInstanceSurvivesEviction(t *testing.T) {
	manager := newTTLInstanceManager(&testInstanceProvider{}, time.Millisecond, time.Millisecond)
	pc := backend.PluginContext{AppInstanceSettings: &backend.AppInstanceSettings{Updated: time.Now()}}
	first, release, err := Acquire(context.Background(), manager, pc)
	require.NoError(t, err)
	defer release()
	old := first.(*testInstance)
	require.Never(t, old.disposed.Load, 20*time.Millisecond, time.Millisecond)
	release()
	require.Eventually(t, func() bool { return old.disposedTimes.Load() == 1 }, time.Second, time.Millisecond)
}

func TestDoReleasesInstanceAfterPanic(t *testing.T) {
	manager := newInstanceManager(&testInstanceProvider{}, time.Millisecond)
	pc := backend.PluginContext{AppInstanceSettings: &backend.AppInstanceSettings{Updated: time.Now()}}
	first, err := manager.Get(context.Background(), pc)
	require.NoError(t, err)
	require.Panics(t, func() {
		_ = manager.Do(context.Background(), pc, func(Instance) { panic("handler panic") })
	})
	pc.AppInstanceSettings = &backend.AppInstanceSettings{Updated: pc.AppInstanceSettings.Updated.Add(time.Second)}
	_, err = manager.Get(context.Background(), pc)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return first.(*testInstance).disposedTimes.Load() == 1 }, time.Second, time.Millisecond)
}
