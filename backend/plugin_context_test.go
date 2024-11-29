package backend_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestPluginContext_GetSettingsFromEnv(t *testing.T) {
	t.Run("should return blank value if no env variables present", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		output := pCtx.GetSettingFromEnv("MY_KEY")
		require.Empty(t, output)
	})
	t.Run("should respect GF_PLUGIN_KEY value", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		t.Setenv("GF_PLUGIN_MY_KEY", "foo")
		output := pCtx.GetSettingFromEnv("MY_KEY")
		require.Equal(t, "foo", output)
	})
	t.Run("should respect GF_PLUGIN_PLUGIN_ID_KEY value", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		t.Setenv("GF_PLUGIN_MY_KEY", "foo")
		t.Setenv("GF_PLUGIN_MY-PLUGIN-ID_MY_KEY", "bar")
		output := pCtx.GetSettingFromEnv("MY_KEY")
		require.Equal(t, "bar", output)
	})
	t.Run("should respect GF_DS_DS_ID_KEY value", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		t.Setenv("GF_PLUGIN_MY_KEY", "foo")
		t.Setenv("GF_PLUGIN_MY-PLUGIN-ID_MY_KEY", "bar")
		t.Setenv("GF_DS_MYDSUID_MY_KEY", "baz")
		output := pCtx.GetSettingFromEnv("MY_KEY")
		require.Equal(t, "baz", output)
	})
	t.Run("should respect case sensitive ds uid", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		t.Setenv("GF_PLUGIN_MY_KEY", "foo")
		t.Setenv("GF_PLUGIN_MY-PLUGIN-ID_MY_KEY", "bar")
		t.Setenv("GF_DS_myDsUid_MY_KEY", "BaZ")
		t.Setenv("GF_DS_MYDSUID_MY_KEY", "baz")
		output := pCtx.GetSettingFromEnv("MY_KEY")
		require.Equal(t, "BaZ", output)
	})
}

func TestPluginContext_GetSettingsAsBoolFromEnv(t *testing.T) {
	t.Run("should return default value when no env variables present", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		output, err := pCtx.GetSettingAsBoolFromEnv("MY_KEY", false)
		require.Nil(t, err)
		require.Equal(t, false, output)
	})
	t.Run("should return default value when no env variables present but default value", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		output, err := pCtx.GetSettingAsBoolFromEnv("MY_KEY", true)
		require.Nil(t, err)
		require.Equal(t, true, output)
	})
	t.Run("should fail with incorrect bool value", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		t.Setenv("GF_PLUGIN_MY_KEY", "foo")
		_, err := pCtx.GetSettingAsBoolFromEnv("MY_KEY", false)
		require.NotNil(t, err)
	})
	t.Run("should parse with correct bool value from environment", func(t *testing.T) {
		pCtx := backend.PluginContext{PluginID: "my-plugin-id", DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{UID: "myDsUid"}}
		t.Setenv("GF_PLUGIN_MY_KEY", "True")
		output, err := pCtx.GetSettingAsBoolFromEnv("MY_KEY", false)
		require.Nil(t, err)
		require.Equal(t, true, output)
	})
}
