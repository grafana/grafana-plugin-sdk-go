package instancemgmt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

type configInstanceProvider struct {
	cached []backend.PluginContext
}

func (p *configInstanceProvider) GetKey(context.Context, backend.PluginContext) (any, error) {
	return "key", nil
}

func (p *configInstanceProvider) NeedsUpdate(_ context.Context, pluginContext backend.PluginContext, cachedInstance CachedInstance) bool {
	p.cached = append(p.cached, cachedInstance.PluginContext)
	return !cachedInstance.PluginContext.GrafanaConfig.Equal(pluginContext.GrafanaConfig)
}

func (p *configInstanceProvider) NewInstance(context.Context, backend.PluginContext) (Instance, error) {
	return &struct{ int }{}, nil
}

func TestCachedInstanceKeepsNoConfigValues(t *testing.T) {
	managers := map[string]func(InstanceProvider) InstanceManager{
		"standard": func(p InstanceProvider) InstanceManager { return newInstanceManager(p, time.Millisecond) },
		"ttl": func(p InstanceProvider) InstanceManager {
			return newTTLInstanceManager(p, time.Hour, time.Hour, time.Millisecond)
		},
	}
	request := func(value string) backend.PluginContext {
		return backend.PluginContext{GrafanaConfig: config.NewGrafanaCfg(map[string]string{"GF_APP_URL": value})}
	}
	for name, newManager := range managers {
		t.Run(name, func(t *testing.T) {
			p := &configInstanceProvider{}
			im := newManager(p)
			ctx := context.Background()

			first, err := im.Get(ctx, request("https://a.example.com"))
			require.NoError(t, err)
			same, err := im.Get(ctx, request("https://a.example.com"))
			require.NoError(t, err)
			require.Same(t, first, same)

			require.NotEmpty(t, p.cached)
			for _, cached := range p.cached {
				require.Empty(t, cached.GrafanaConfig.Config())
				require.Empty(t, cached.GrafanaConfig.Get("GF_APP_URL"))
			}

			changed, err := im.Get(ctx, request("https://b.example.com"))
			require.NoError(t, err)
			require.NotSame(t, first, changed)
		})
	}
}
