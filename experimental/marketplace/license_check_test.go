package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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

func TestInvalidLicenseHandlerCheckHealthJSONDetails(t *testing.T) {
	message := "message with \"quotes\", \\backslashes\\, and\na newline"
	verboseMessage := "verbose with \"quotes\", \\backslashes\\, and\na newline"
	handler := invalidLicenseHandler{
		err:          errors.New(message),
		verboseError: errors.New(verboseMessage),
	}

	result, err := handler.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	require.NoError(t, err)
	require.True(t, json.Valid(result.JSONDetails))

	var details struct {
		Message        string `json:"message"`
		VerboseMessage string `json:"verboseMessage"`
	}
	require.NoError(t, json.Unmarshal(result.JSONDetails, &details))
	require.Equal(t, message, details.Message)
	require.Equal(t, strings.ReplaceAll(verboseMessage, "\n", " "), details.VerboseMessage)
}
