package datasource

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
	ip := NewInstanceProvider(func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		return testInstance{value: "hello"}, nil
	})

	t.Run("When data source instance settings not provided should return error", func(t *testing.T) {
		_, err := ip.GetKey(backend.PluginContext{})
		require.Error(t, err)
	})

	t.Run("When data source instance settings provided should return expected key", func(t *testing.T) {
		key, err := ip.GetKey(backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
				ID: 4,
			},
		})
		require.NoError(t, err)
		require.Equal(t, int64(4), key)
	})

	t.Run("When current data source instance settings compared to cached instance haven't been updated should return false", func(t *testing.T) {
		curSettings := backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
				Updated: time.Now(),
			},
		}
		cachedInstance := instancemgmt.CachedInstance{
			PluginContext: backend.PluginContext{
				DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
					Updated: curSettings.DataSourceInstanceSettings.Updated,
				},
			},
		}
		needsUpdate := ip.NeedsUpdate(curSettings, cachedInstance)
		require.False(t, needsUpdate)
	})

	t.Run("When current data source instance settings compared to cached instance have been updated should return true", func(t *testing.T) {
		curSettings := backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
				Updated: time.Now(),
			},
		}
		cachedInstance := instancemgmt.CachedInstance{
			PluginContext: backend.PluginContext{
				DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
					Updated: curSettings.DataSourceInstanceSettings.Updated.Add(time.Second),
				},
			},
		}
		needsUpdate := ip.NeedsUpdate(curSettings, cachedInstance)
		require.True(t, needsUpdate)
	})

	t.Run("When creating a new instance should return expected instance", func(t *testing.T) {
		i, err := ip.NewInstance(backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		})
		require.NoError(t, err)
		require.NotNil(t, i)
		require.Equal(t, "hello", i.(testInstance).value)
	})
}
