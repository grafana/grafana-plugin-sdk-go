package instancemgmt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func TestInstanceManagerWrapper(t *testing.T) {
	ctx := context.Background()
	tip := &testInstanceProvider{}
	im := NewInstanceManagerWrapper(tip)

	t.Run("Should use standard manager when feature toggle is disabled", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: "",
			}),
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManager{}, manager)
	})

	t.Run("Should use TTL manager when feature toggle is enabled", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: featuretoggles.TTLInstanceManager,
			}),
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManagerWithTTL{}, manager)
	})

	t.Run("Should use standard manager when GrafanaConfig is nil", func(t *testing.T) {
		pCtx := backend.PluginContext{
			OrgID: 1,
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
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: "someOtherFlag," + featuretoggles.TTLInstanceManager + ",anotherFlag",
			}),
		}

		manager := im.(*instanceManagerWrapper).selectManager(ctx, pCtx)
		require.IsType(t, &instanceManagerWithTTL{}, manager)
	})

	t.Run("Should delegate Get calls correctly", func(t *testing.T) {
		// Test with TTL manager enabled
		pCtx := backend.PluginContext{
			OrgID: 1,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: featuretoggles.TTLInstanceManager,
			}),
		}

		instance, err := im.Get(ctx, pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)
		require.Equal(t, pCtx.OrgID, instance.(*testInstance).orgID)
	})

	t.Run("Should delegate Do calls correctly", func(t *testing.T) {
		// Test with standard manager (no feature toggle)
		pCtx := backend.PluginContext{
			OrgID: 2,
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
			GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
				featuretoggles.EnabledFeatures: "",
			}),
		}

		var receivedInstance *testInstance
		err := im.Do(ctx, pCtx, func(instance Instance) {
			receivedInstance = instance.(*testInstance)
		})
		require.NoError(t, err)
		require.NotNil(t, receivedInstance)
		require.Equal(t, pCtx.OrgID, receivedInstance.orgID)
	})
}
