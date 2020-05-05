package instancemgmt

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestInstanceManager(t *testing.T) {
	pCtx := backend.PluginContext{
		OrgID: 1,
		AppInstanceSettings: &backend.AppInstanceSettings{
			Updated: time.Now(),
		},
	}

	tip := &testInstanceProvider{}
	im := New(tip)

	t.Run("When getting instance should create a new instance", func(t *testing.T) {
		instance, err := im.Get(pCtx)
		require.NoError(t, err)
		require.NotNil(t, instance)
		require.Equal(t, pCtx.OrgID, instance.(*testInstance).orgID)
		require.Equal(t, pCtx.AppInstanceSettings.Updated, instance.(*testInstance).updated)

		t.Run("When getting instance should return same instance", func(t *testing.T) {
			instance2, err := im.Get(pCtx)
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
			newInstance, err := im.Get(pCtxUpdated)

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
				require.True(t, instance.(*testInstance).disposed)
			})
		})
	})
}

type testInstance struct {
	orgID    int64
	updated  time.Time
	disposed bool
}

func (ti *testInstance) Dispose() {
	ti.disposed = true
}

type testInstanceProvider struct {
}

func (tip *testInstanceProvider) GetKey(pluginContext backend.PluginContext) (interface{}, error) {
	return pluginContext.OrgID, nil
}

func (tip *testInstanceProvider) NeedsUpdate(pluginContext backend.PluginContext, cachedInstance CachedInstance) bool {
	curUpdated := pluginContext.AppInstanceSettings.Updated
	cachedUpdated := cachedInstance.PluginContext.AppInstanceSettings.Updated
	return !curUpdated.Equal(cachedUpdated)
}

func (tip *testInstanceProvider) NewInstance(pluginContext backend.PluginContext) (Instance, error) {
	return &testInstance{
		orgID:   pluginContext.OrgID,
		updated: pluginContext.AppInstanceSettings.Updated,
	}, nil
}
