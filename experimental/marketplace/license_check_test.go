package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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

func TestMarketplaceLicenseGracePeriodUsesLicenseExpiration(t *testing.T) {
	now := time.Date(2025, time.January, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		tokenExpires   time.Time
		licenseExpires time.Time
		wantWithin     bool
	}{
		{
			// This case is important for on-prem instances that lost internet connection and cannot
			// renew the token. Air-gapped by contract can have Expires = TokenExpires, non-air-gapped
			// by contract may lose connection at any time, so we have to account for that (allow to run
			// even if the token is technically expires, but the entitlement hasn't expired yet).
			// If the entitlement expires, the administrator must pay and then restore internet access
			// (or re-provision via disk, like for air-gapped) for the plugin to continue to work.
			name:           "valid license grace period despite an old token expiration",
			tokenExpires:   now.Add(-30 * 24 * time.Hour),
			licenseExpires: now.Add(-24 * time.Hour),
			wantWithin:     true,
		},
		{
			name:           "expired license grace period despite a recent token expiration",
			tokenExpires:   now.Add(-24 * time.Hour),
			licenseExpires: now.Add(-30 * 24 * time.Hour),
			wantWithin:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := &licensing.LicenseToken{
				Expires:        tc.tokenExpires.Unix(),
				LicenseExpires: tc.licenseExpires.Unix(),
			}

			grace, within := marketplaceLicenseGracePeriod(token, now)

			require.True(t, tc.licenseExpires.Add(14*24*time.Hour).Equal(grace))
			require.Equal(t, tc.wantWithin, within)
		})
	}
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

func newTestInvalidLicenseHandler() *invalidLicenseHandler {
	return &invalidLicenseHandler{
		pluginID:     "example-app",
		err:          invalidLicenseError("example-app"),
		verboseError: errors.New("license token file not found"),
	}
}

func TestInvalidLicenseHandlerCallResource(t *testing.T) {
	handler := newTestInvalidLicenseHandler()
	var resp *backend.CallResourceResponse
	err := handler.CallResource(context.Background(), &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(r *backend.CallResourceResponse) error {
		resp = r
		return nil
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusForbidden, resp.Status)
	require.Equal(t, []string{"text/plain; charset=utf-8"}, resp.Headers["Content-Type"])
	require.Equal(t, handler.err.Error(), string(resp.Body))
}

func TestInvalidLicenseHandlerQueryData(t *testing.T) {
	handler := newTestInvalidLicenseHandler()
	resp, err := handler.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{{RefID: "A"}, {RefID: "B"}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Responses, 2)
	for _, refID := range []string{"A", "B"} {
		require.Equal(t, backend.StatusUnauthorized, resp.Responses[refID].Status)
		require.ErrorContains(t, resp.Responses[refID].Error, handler.err.Error())
	}
}

type recordingChunkedDataWriter struct {
	backend.ChunkedDataWriter
	errors map[string]error
}

func (w *recordingChunkedDataWriter) WriteError(_ context.Context, refID string, status backend.Status, err error) error {
	if status != backend.StatusUnauthorized {
		return fmt.Errorf("unexpected status %v", status)
	}
	w.errors[refID] = err
	return nil
}

func TestInvalidLicenseHandlerQueryChunkedData(t *testing.T) {
	handler := newTestInvalidLicenseHandler()
	w := &recordingChunkedDataWriter{errors: map[string]error{}}
	err := handler.QueryChunkedData(context.Background(), &backend.QueryChunkedDataRequest{
		Queries: []backend.DataQuery{{RefID: "A"}, {RefID: "B"}},
	}, w)
	require.NoError(t, err)
	require.Len(t, w.errors, 2)
	for _, refID := range []string{"A", "B"} {
		require.ErrorContains(t, w.errors[refID], handler.err.Error())
	}
}

func TestInvalidLicenseHandlerStreams(t *testing.T) {
	handler := newTestInvalidLicenseHandler()

	subscribe, err := handler.SubscribeStream(context.Background(), &backend.SubscribeStreamRequest{})
	require.NoError(t, err)
	require.Equal(t, backend.SubscribeStreamStatusPermissionDenied, subscribe.Status)

	publish, err := handler.PublishStream(context.Background(), &backend.PublishStreamRequest{})
	require.NoError(t, err)
	require.Equal(t, backend.PublishStreamStatusPermissionDenied, publish.Status)

	require.ErrorIs(t, handler.RunStream(context.Background(), &backend.RunStreamRequest{}, nil), handler.err)
}
