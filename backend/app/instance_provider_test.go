package app

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/stretchr/testify/require"
)

func TestInstanceProvider(t *testing.T) {
	type testInstance struct {
		value string
	}
	ip := NewInstanceProvider(func(settings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
		return testInstance{value: "hello"}, nil
	})

	t.Run("When application instance settings not provided should return error", func(t *testing.T) {
		_, err := ip.GetKey(backend.PluginContext{})
		require.Error(t, err)
	})

	t.Run("When application instance settings provided should return expected key", func(t *testing.T) {
		key, err := ip.GetKey(backend.PluginContext{
			OrgID:               2,
			AppInstanceSettings: &backend.AppInstanceSettings{},
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), key)
	})

	t.Run("When current application instance settings compared to cached instance haven't been updated should return false", func(t *testing.T) {
		curSettings := backend.PluginContext{
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
		}
		cachedInstance := instancemgmt.CachedInstance{
			PluginContext: backend.PluginContext{
				AppInstanceSettings: &backend.AppInstanceSettings{
					Updated: curSettings.AppInstanceSettings.Updated,
				},
			},
		}
		needsUpdate := ip.NeedsUpdate(curSettings, cachedInstance)
		require.False(t, needsUpdate)
	})

	t.Run("When current application instance settings compared to cached instance have been updated should return true", func(t *testing.T) {
		curSettings := backend.PluginContext{
			AppInstanceSettings: &backend.AppInstanceSettings{
				Updated: time.Now(),
			},
		}
		cachedInstance := instancemgmt.CachedInstance{
			PluginContext: backend.PluginContext{
				AppInstanceSettings: &backend.AppInstanceSettings{
					Updated: curSettings.AppInstanceSettings.Updated.Add(time.Second),
				},
			},
		}
		needsUpdate := ip.NeedsUpdate(curSettings, cachedInstance)
		require.True(t, needsUpdate)
	})

	t.Run("When creating a new instance should return expected instance", func(t *testing.T) {
		i, err := ip.NewInstance(backend.PluginContext{
			AppInstanceSettings: &backend.AppInstanceSettings{},
		})
		require.NoError(t, err)
		require.NotNil(t, i)
		require.Equal(t, "hello", i.(testInstance).value)
	})
}
