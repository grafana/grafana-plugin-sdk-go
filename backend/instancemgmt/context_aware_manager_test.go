package instancemgmt

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func TestInstanceManagerWrapper(t *testing.T) {
	ctx := context.Background()
	tip := &testInstanceProvider{}
	im := NewInstanceManagerWrapper(tip)

	t.Run("Should use standard manager when feature toggle is disabled", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1, // nolint:staticcheck
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: config.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: "",
			}),
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManager{}, manager)
	})

	t.Run("Should use TTL manager when feature toggle is enabled", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1, // nolint:staticcheck
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: config.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: featuretoggles.TTLInstanceManager,
			}),
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManagerWithTTL{}, manager)
	})

	t.Run("Should use standard manager when GrafanaConfig is nil", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1, // nolint:staticcheck
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: nil,
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManager{}, manager)
	})

	t.Run("Should use TTL manager when feature toggle is enabled with other flags", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1, // nolint:staticcheck
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: config.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: "someOtherFlag," + featuretoggles.TTLInstanceManager + ",anotherFlag",
			}),
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManagerWithTTL{}, manager)
	})

	t.Run("Should delegate Get calls correctly", func(t *testing.T) {
		// Test with TTL manager enabled
		pCtx := backend.PluginContext{
			OrgID: 1, // nolint:staticcheck
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: config.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: featuretoggles.TTLInstanceManager,
			}),
		}

		instance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)
		require.Equal(t, pCtx.OrgID, instance.(*testInstance).orgID) // nolint:staticcheck
	})

	t.Run("Should delegate Do calls correctly", func(t *testing.T) {
		// Test with standard manager (no feature toggle)
		pCtx := backend.PluginContext{
			OrgID: 2, // nolint:staticcheck
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: config.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: "",
			}),
		}

		var receivedInstance *testInstance
		err := im.Do(ctx, pCtx, func(instance Instance) {
			receivedInstance = instance.(*testInstance)
		})
		require.NoError(t, err)
		require.NotNil(t, receivedInstance)
		require.Equal(t, pCtx.OrgID, receivedInstance.orgID) // nolint:staticcheck
	})
}

func TestInstanceManagerWrapper_LazilyConstructsTTLManager(t *testing.T) {
	ctx := context.Background()
	tip := &testInstanceProvider{}

	runtime.Gosched()
	before := runtime.NumGoroutine()

	im := NewInstanceManagerWrapper(tip)

	// The TTL manager's own cache spawns a background cleanup goroutine on
	// construction. It should not be constructed at all until a plugin
	// context actually selects it, so simply creating the wrapper -- as
	// every datasource.Manage-based plugin does at startup, regardless of
	// whether the TTL toggle is ever enabled for it -- must not spawn one.
	time.Sleep(50 * time.Millisecond)
	afterConstruction := runtime.NumGoroutine()
	require.Equal(t, before, afterConstruction,
		"constructing the instance manager wrapper should not spawn the TTL manager's cleanup goroutine")

	pCtx := backend.PluginContext{
		OrgID: 1, // nolint:staticcheck
		AppInstanceSettings: &backend.AppInstanceSettings{
			Updated: time.Now(),
		},
		GrafanaConfig: config.NewGrafanaCfg(map[string]string{
			featuretoggles.EnabledFeatures: featuretoggles.TTLInstanceManager,
		}),
	}

	// Selecting the TTL manager repeatedly should construct it -- and its
	// cleanup goroutine -- exactly once, not once per call.
	for range 10 {
		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManagerWithTTL{}, manager)
	}

	time.Sleep(50 * time.Millisecond)
	afterSelect := runtime.NumGoroutine()
	require.Equal(t, before+1, afterSelect,
		"selecting the TTL manager should construct its cleanup goroutine exactly once, regardless of how many times it's selected")
}
