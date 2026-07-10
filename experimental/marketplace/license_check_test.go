package marketplace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
	"github.com/stretchr/testify/require"
)

func TestReadPluginLicenseFallbackFileName(t *testing.T) {
	const pluginID = "test-plugin"
	const licenseFileName = "license-" + pluginID + ".jwt"

	t.Setenv(marketplaceLicenseTextEnv, "")
	t.Setenv(marketplaceLicensePathEnv, "")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	workingDir := t.TempDir()
	originalWorkingDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workingDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWorkingDir))
	})

	require.NoError(t, os.WriteFile(licenseFileName, nil, 0o600))
	require.Equal(t, licensing.Invalid, readPluginLicense(pluginID).Status)

	require.NoError(t, os.Remove(licenseFileName))
	token := readPluginLicense(pluginID)
	require.Equal(t, licensing.NotFound, token.Status)
	require.Equal(t, "license token file not found: "+filepath.Join(homeDir, ".grafana", licenseFileName), token.Error.Error())
}
