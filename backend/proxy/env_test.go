package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientCfgFromEnv(t *testing.T) {
	// create empty file for testing configs
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test")
	testFileData := "foobar"
	err := os.WriteFile(testFilePath, []byte(testFileData), 0600)
	require.NoError(t, err)

	cases := []struct {
		description string
		envVars     map[string]string
		expected    *ClientCfg
	}{
		{
			description: "socks proxy not enabled, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:             "false",
				PluginSecureSocksProxyAddressEnvVarName:             "localhost",
				PluginSecureSocksProxyClientCertFilePathEnvVarName:  "cert",
				PluginSecureSocksProxyClientKeyFilePathEnvVarName:   "key",
				PluginSecureSocksProxyRootCACertFilePathsEnvVarName: "root_ca",
				PluginSecureSocksProxyServerNameEnvVarName:          "server_name",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=true, should return config without tls fields filled",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:       "true",
				PluginSecureSocksProxyAddressEnvVarName:       "localhost",
				PluginSecureSocksProxyAllowInsecureEnvVarName: "true",
			},
			expected: &ClientCfg{
				ProxyAddress:  "localhost",
				AllowInsecure: true,
			},
		},
		{
			description: "allowInsecure=false, client cert is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:       "true",
				PluginSecureSocksProxyAddressEnvVarName:       "localhost",
				PluginSecureSocksProxyAllowInsecureEnvVarName: "false",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, client key is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:            "true",
				PluginSecureSocksProxyAddressEnvVarName:            "localhost",
				PluginSecureSocksProxyAllowInsecureEnvVarName:      "false",
				PluginSecureSocksProxyClientCertFilePathEnvVarName: "cert",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, root ca is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:            "true",
				PluginSecureSocksProxyAddressEnvVarName:            "localhost",
				PluginSecureSocksProxyAllowInsecureEnvVarName:      "false",
				PluginSecureSocksProxyClientCertFilePathEnvVarName: "cert",
				PluginSecureSocksProxyClientKeyFilePathEnvVarName:  "key",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, server name is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:             "true",
				PluginSecureSocksProxyAddressEnvVarName:             "localhost",
				PluginSecureSocksProxyAllowInsecureEnvVarName:       "false",
				PluginSecureSocksProxyClientCertFilePathEnvVarName:  "cert",
				PluginSecureSocksProxyClientKeyFilePathEnvVarName:   "key",
				PluginSecureSocksProxyRootCACertFilePathsEnvVarName: "root",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, should return config with tls fields filled",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabledEnvVarName:             "true",
				PluginSecureSocksProxyAddressEnvVarName:             "localhost",
				PluginSecureSocksProxyAllowInsecureEnvVarName:       "false",
				PluginSecureSocksProxyClientCertFilePathEnvVarName:  testFilePath,
				PluginSecureSocksProxyClientKeyFilePathEnvVarName:   testFilePath,
				PluginSecureSocksProxyRootCACertFilePathsEnvVarName: fmt.Sprintf("%s %s", testFilePath, testFilePath),
				PluginSecureSocksProxyServerNameEnvVarName:          "name",
			},
			expected: &ClientCfg{
				ProxyAddress:  "localhost",
				ClientCertVal: testFileData,
				ClientKeyVal:  testFileData,
				RootCAsVals:   []string{testFileData, testFileData},
				ServerName:    "name",
				AllowInsecure: false,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}
			assert.Equal(t, tt.expected, clientCfgFromEnv())
		})
	}
}
