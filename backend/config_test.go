package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-azure-sdk-go/azsettings"
	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func TestConfig(t *testing.T) {
	t.Run("GrafanaConfigFromContext", func(t *testing.T) {
		tcs := []struct {
			name                   string
			cfg                    *GrafanaCfg
			expectedFeatureToggles FeatureToggles
			expectedProxy          Proxy
			expectedAzure          *azsettings.AzureSettings
		}{
			{
				name:                   "nil config",
				cfg:                    nil,
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
				expectedAzure:          &azsettings.AzureSettings{},
			},
			{
				name:                   "empty config",
				cfg:                    &GrafanaCfg{},
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
				expectedAzure:          &azsettings.AzureSettings{},
			},
			{
				name:                   "nil config map",
				cfg:                    NewGrafanaCfg(nil),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
				expectedAzure:          &azsettings.AzureSettings{},
			},
			{
				name:                   "empty config map",
				cfg:                    NewGrafanaCfg(make(map[string]string)),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
				expectedAzure:          &azsettings.AzureSettings{},
			},
			{
				name: "feature toggles and proxy enabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "TestFeature",
					proxy.PluginSecureSocksProxyEnabled:      "true",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:   "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{
					enabled: map[string]struct{}{
						"TestFeature": {},
					},
				},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
						ClientCert:   "clientCert",
						ClientKey:    "clientKey",
						RootCA:       "rootCACert",
						ProxyAddress: "localhost:1234",
						ServerName:   "localhost",
					},
				},
				expectedAzure: &azsettings.AzureSettings{},
			},
			{
				name: "feature toggles enabled and proxy disabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "TestFeature",
					proxy.PluginSecureSocksProxyEnabled:      "false",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:   "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{
					enabled: map[string]struct{}{
						"TestFeature": {},
					},
				},
				expectedProxy: Proxy{},
				expectedAzure: &azsettings.AzureSettings{},
			},
			{
				name: "feature toggles disabled and proxy enabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "",
					proxy.PluginSecureSocksProxyEnabled:      "true",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:   "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
						ClientCert:   "clientCert",
						ClientKey:    "clientKey",
						RootCA:       "rootCACert",
						ProxyAddress: "localhost:1234",
						ServerName:   "localhost",
					},
				},
				expectedAzure: &azsettings.AzureSettings{},
			},
			{
				name: "azure settings in config",
				cfg: NewGrafanaCfg(map[string]string{
					azsettings.AzureCloud:                azsettings.AzureCloud,
					azsettings.ManagedIdentityEnabled:    "true",
					azsettings.ManagedIdentityClientID:   "mock_managed_identity_client_id",
					azsettings.UserIdentityEnabled:       "true",
					azsettings.UserIdentityClientID:      "mock_user_identity_client_id",
					azsettings.UserIdentityClientSecret:  "mock_managed_identity_client_secret",
					azsettings.UserIdentityTokenURL:      "mock_managed_identity_token_url",
					azsettings.UserIdentityAssertion:     "username",
					azsettings.WorkloadIdentityEnabled:   "true",
					azsettings.WorkloadIdentityClientID:  "mock_workload_identity_client_id",
					azsettings.WorkloadIdentityTenantID:  "mock_workload_identity_tenant_id",
					azsettings.WorkloadIdentityTokenFile: "mock_workload_identity_token_file",
				}),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
				expectedAzure: &azsettings.AzureSettings{
					Cloud:                   azsettings.AzureCloud,
					ManagedIdentityEnabled:  true,
					ManagedIdentityClientId: "mock_managed_identity_client_id",
					UserIdentityEnabled:     true,
					UserIdentityTokenEndpoint: &azsettings.TokenEndpointSettings{
						ClientId:          "mock_user_identity_client_id",
						ClientSecret:      "mock_managed_identity_client_secret",
						TokenUrl:          "mock_managed_identity_token_url",
						UsernameAssertion: true,
					},
					WorkloadIdentityEnabled: true,
					WorkloadIdentitySettings: &azsettings.WorkloadIdentitySettings{
						ClientId:  "mock_workload_identity_client_id",
						TenantId:  "mock_workload_identity_tenant_id",
						TokenFile: "mock_workload_identity_token_file",
					},
				},
			},
		}

		for _, tc := range tcs {
			ctx := WithGrafanaConfig(context.Background(), tc.cfg)
			cfg := GrafanaConfigFromContext(ctx)

			require.Equal(t, tc.expectedFeatureToggles, cfg.FeatureToggles())
			require.Equal(t, tc.expectedProxy, cfg.proxy())
			require.Equal(t, tc.expectedAzure, cfg.Azure())
		}
	})
}

func TestAppURL(t *testing.T) {
	t.Run("it should return the configured app URL", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			AppURL: "http://localhost:3000",
		})
		url, err := cfg.AppURL()
		require.NoError(t, err)
		require.Equal(t, "http://localhost:3000", url)
	})

	t.Run("it should return an error if the app URL is missing", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		_, err := cfg.AppURL()
		require.Error(t, err)
	})
}

func TestUserAgentFromContext(t *testing.T) {
	ua, err := useragent.New("10.0.0", "test", "test")
	require.NoError(t, err)

	ctx := WithUserAgent(context.Background(), ua)
	result := UserAgentFromContext(ctx)

	require.Equal(t, "10.0.0", result.GrafanaVersion())
	require.Equal(t, "Grafana/10.0.0 (test; test)", result.String())
}

func TestUserAgentFromContext_NoUserAgent(t *testing.T) {
	ctx := context.Background()

	result := UserAgentFromContext(ctx)
	require.Equal(t, "0.0.0", result.GrafanaVersion())
	require.Equal(t, "Grafana/0.0.0 (unknown; unknown)", result.String())
}
