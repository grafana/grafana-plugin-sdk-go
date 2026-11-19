package datasource

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func TestInstanceManagerIgnoresUnusedPDCFilePaths(t *testing.T) {
	toggles := map[string]string{"standard": "", "ttl": featuretoggles.TTLInstanceManager}
	for name, enabled := range toggles {
		t.Run(name, func(t *testing.T) {
			var created atomic.Int32
			im := NewInstanceManager(func(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
				created.Add(1)
				return &struct{ int }{}, nil
			})
			request := func(paths, rootCA string) backend.PluginContext {
				return backend.PluginContext{
					DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "uid"},
					GrafanaConfig: config.NewGrafanaCfg(map[string]string{
						featuretoggles.EnabledFeatures:                 enabled,
						proxy.PluginSecureSocksProxyClientCertContents: "cert",
						proxy.PluginSecureSocksProxyRootCAsContents:    rootCA,
						proxy.PluginSecureSocksProxyRootCAs:            paths + "/ca.crt",     // nolint:staticcheck
						proxy.PluginSecureSocksProxyClientCert:         paths + "/client.crt", // nolint:staticcheck
					}),
				}
			}
			get := func(pCtx backend.PluginContext) instancemgmt.Instance {
				t.Helper()
				instance, err := im.Get(config.WithGrafanaConfig(context.Background(), pCtx.GrafanaConfig), pCtx)
				require.NoError(t, err)
				return instance
			}

			first := get(request("/certs", "ca"))
			require.Same(t, first, get(request("/other", "ca")))
			require.Equal(t, int32(1), created.Load())

			require.NotSame(t, first, get(request("/other", "other-ca")))
			require.Equal(t, int32(2), created.Load())
		})
	}
}
