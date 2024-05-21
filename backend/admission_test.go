package backend_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestInstanceSettingsAdmissionConversions(t *testing.T) {
	t.Run("DataSource", func(t *testing.T) {
		before := &backend.DataSourceInstanceSettings{
			URL:      "http://something",
			Updated:  time.Now(),
			User:     "u",
			JSONData: []byte(`{"hello": "world"}`),
			DecryptedSecureJSONData: map[string]string{
				"A": "B",
			},
		}
		req := before.ToAdmissionRequest(nil)
		after, err := backend.DataSourceInstanceSettingsFromProto(req.ObjectBytes, "")
		require.NoError(t, err)
		require.Equal(t, before.URL, after.URL)
		require.Equal(t, before.User, after.User)
		require.Equal(t, before.JSONData, after.JSONData)
		require.Equal(t, before.DecryptedSecureJSONData, after.DecryptedSecureJSONData)
	})

	t.Run("App", func(t *testing.T) {
		before := &backend.AppInstanceSettings{
			Updated:  time.Now(),
			JSONData: []byte(`{"hello": "world"}`),
			DecryptedSecureJSONData: map[string]string{
				"A": "B",
			},
		}
		req := before.ToAdmissionRequest(nil)
		after, err := backend.AppInstanceSettingsFromProto(req.ObjectBytes)
		require.NoError(t, err)
		require.Equal(t, before.JSONData, after.JSONData)
		require.Equal(t, before.DecryptedSecureJSONData, after.DecryptedSecureJSONData)
	})
}
