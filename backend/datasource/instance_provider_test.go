package datasource

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/stretchr/testify/require"
)

func TestInstanceProvider(t *testing.T) {
	type testInstance struct {
		value string
	}
	ip := NewInstanceProvider(func(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		return testInstance{value: "hello"}, nil
	})

	t.Run("When data source instance settings not provided should return error", func(t *testing.T) {
		_, err := ip.GetKey(context.Background(), backend.PluginContext{})
		require.Error(t, err)
	})

	t.Run("When data source instance settings provided should return expected key", func(t *testing.T) {
		key, err := ip.GetKey(context.Background(), backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
				ID: 4,
			},
		})
		require.NoError(t, err)
		require.Equal(t, "4#", key)
	})

	t.Run("When creating a new instance should return expected instance", func(t *testing.T) {
		i, err := ip.NewInstance(context.Background(), backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		})
		require.NoError(t, err)
		require.NotNil(t, i)
		require.Equal(t, "hello", i.(testInstance).value)
	})
}

func Test_instanceProvider_NeedsUpdate(t *testing.T) {
	ts := time.Now()

	type args struct {
		pluginContext backend.PluginContext
		cachedContext backend.PluginContext
	}
	tests := []struct {
		name     string
		args     args
		expected bool
	}{
		{
			name: "Empty instance settings should return false",
			args: args{
				pluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
				},
				cachedContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
				},
			},
			expected: false,
		},
		{
			name: "Instance settings with identical updated field should return false",
			args: args{
				pluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
				},
				cachedContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
				},
			},
			expected: false,
		},
		{
			name: "Instance settings with identical updated field and config should return false",
			args: args{
				pluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
					GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
						"foo": "bar",
						"baz": "qux",
					}),
				},
				cachedContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
					GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
						"baz": "qux",
						"foo": "bar",
					}),
				},
			},
			expected: false,
		},
		{
			name: "Instance settings with different updated field should return true",
			args: args{
				pluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
				},
				cachedContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts.Add(time.Millisecond),
					},
				},
			},
			expected: true,
		},
		{
			name: "Instance settings with identical updated field and different config should return true",
			args: args{
				pluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
					GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
						"foo": "bar",
					}),
				},
				cachedContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
				},
			},
			expected: true,
		},
		{
			name: "Instance settings with identical updated field and config different only by volatile keys should return false",
			args: args{
				pluginContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
					GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
						proxy.PluginSecureSocksProxyClientCertContents: "rhinoceros",
						proxy.PluginSecureSocksProxyClientKeyContents:  "purring",
					}),
				},
				cachedContext: backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
						Updated: ts,
					},
					GrafanaConfig: backend.NewGrafanaCfg(map[string]string{
						proxy.PluginSecureSocksProxyClientCertContents: "elephant",
						proxy.PluginSecureSocksProxyClientKeyContents:  "howling",
					}),
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &instanceProvider{}
			cached := instancemgmt.CachedInstance{PluginContext: tt.args.cachedContext}
			if got := ip.NeedsUpdate(context.Background(), tt.args.pluginContext, cached); got != tt.expected {
				t.Errorf("NeedsUpdate() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
