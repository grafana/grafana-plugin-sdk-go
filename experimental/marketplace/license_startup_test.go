package marketplace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
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

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "license.jwt")
	require.NoError(t, os.WriteFile(file, nil, 0o600))

	require.True(t, fileExists(file))
	require.False(t, fileExists(tempDir))
	require.False(t, fileExists("\x00"))
}

func TestReadPluginLicenseConfiguredSingleCharacterPath(t *testing.T) {
	t.Setenv(marketplaceLicenseTextEnv, "")
	t.Setenv(marketplaceLicensePathEnv, "a")

	originalWorkingDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWorkingDir))
	})

	token := readPluginLicense("test-plugin")
	require.Equal(t, licensing.NotFound, token.Status)
	require.EqualError(t, token.Error, "license token file not found: a")
}
