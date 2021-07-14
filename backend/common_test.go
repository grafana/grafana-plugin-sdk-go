package backend

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceInstanceSettings(t *testing.T) {
	t.Run("HTTPClientOptions() should translate basic auth settings as expected", func(t *testing.T) {
		tcs := []struct {
			instanceSettings      *DataSourceInstanceSettings
			expectedClientOptions httpclient.Options
		}{
			{
				instanceSettings:      &DataSourceInstanceSettings{},
				expectedClientOptions: httpclient.Options{},
			},
			{
				instanceSettings: &DataSourceInstanceSettings{
					User:             "user",
					JSONData:         []byte("{}"),
					BasicAuthEnabled: true,
					BasicAuthUser:    "buser",
					DecryptedSecureJSONData: map[string]string{
						"basicAuthPassword": "bpwd",
						"password":          "pwd",
					},
				},
				expectedClientOptions: httpclient.Options{
					BasicAuth: &httpclient.BasicAuthOptions{
						User:     "buser",
						Password: "bpwd",
					},
				},
			},
			{
				instanceSettings: &DataSourceInstanceSettings{
					User:             "user",
					JSONData:         []byte("{}"),
					BasicAuthEnabled: false,
					BasicAuthUser:    "buser",
					DecryptedSecureJSONData: map[string]string{
						"basicAuthPassword": "bpwd",
						"password":          "pwd",
					},
				},
				expectedClientOptions: httpclient.Options{
					BasicAuth: &httpclient.BasicAuthOptions{
						User:     "user",
						Password: "pwd",
					},
				},
			},
		}

		for _, tc := range tcs {
			opts, err := tc.instanceSettings.HTTPClientOptions()
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedClientOptions.BasicAuth, opts.BasicAuth)
		}
	})
}
