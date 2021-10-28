package backend

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppInstanceSettings(t *testing.T) {
	t.Run("HTTPClientOptions() should translate settings as expected", func(t *testing.T) {
		tcs := []struct {
			instanceSettings      *AppInstanceSettings
			expectedClientOptions httpclient.Options
		}{
			{
				instanceSettings:      &AppInstanceSettings{},
				expectedClientOptions: httpclient.Options{},
			},
			{
				instanceSettings: &AppInstanceSettings{
					JSONData: []byte("{ \"key\": \"value\" }"),
					DecryptedSecureJSONData: map[string]string{
						"sKey": "sValue",
					},
				},
				expectedClientOptions: httpclient.Options{
					CustomOptions: map[string]interface{}{
						dataCustomOptionsKey: map[string]interface{}{
							"key": "value",
						},
						secureDataCustomOptionsKey: map[string]string{
							"sKey": "sValue",
						},
					},
				},
			},
		}

		for _, tc := range tcs {
			opts, err := tc.instanceSettings.HTTPClientOptions()
			assert.NoError(t, err)
			if tc.expectedClientOptions.BasicAuth != nil {
				assert.Equal(t, tc.expectedClientOptions.BasicAuth, opts.BasicAuth)
			} else {
				assert.Nil(t, opts.BasicAuth)
			}

			if tc.expectedClientOptions.Labels != nil {
				assert.Equal(t, tc.expectedClientOptions.Labels, opts.Labels)
			}

			jsonData := JSONDataFromHTTPClientOptions(opts)
			expectedJSONData := JSONDataFromHTTPClientOptions(tc.expectedClientOptions)
			secureJSONData := SecureJSONDataFromHTTPClientOptions(opts)
			expectedSecureJSONData := SecureJSONDataFromHTTPClientOptions(tc.expectedClientOptions)

			if len(tc.expectedClientOptions.CustomOptions) > 0 {
				assert.Equal(t, tc.expectedClientOptions.CustomOptions, opts.CustomOptions)
				assert.Equal(t, expectedJSONData, jsonData)
				assert.Equal(t, expectedSecureJSONData, secureJSONData)
			} else {
				assert.Empty(t, opts.CustomOptions)
				assert.Empty(t, jsonData)
				assert.Empty(t, secureJSONData)
			}
		}
	})
}

func TestDataSourceInstanceSettings(t *testing.T) {
	t.Run("HTTPClientOptions() should translate settings as expected", func(t *testing.T) {
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
					Name:             "ds1",
					UID:              "uid1",
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
					Labels: map[string]string{
						"datasource_name": "ds1",
						"datasource_uid":  "uid1",
					},
					CustomOptions: map[string]interface{}{
						dataCustomOptionsKey: map[string]interface{}{},
						secureDataCustomOptionsKey: map[string]string{
							"basicAuthPassword": "bpwd",
							"password":          "pwd",
						},
					},
				},
			},
			{
				instanceSettings: &DataSourceInstanceSettings{
					Name:             "ds2",
					UID:              "uid2",
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
					Labels: map[string]string{
						"datasource_name": "ds2",
						"datasource_uid":  "uid2",
					},
					CustomOptions: map[string]interface{}{
						dataCustomOptionsKey: map[string]interface{}{},
						secureDataCustomOptionsKey: map[string]string{
							"basicAuthPassword": "bpwd",
							"password":          "pwd",
						},
					},
				},
			},
			{
				instanceSettings: &DataSourceInstanceSettings{
					JSONData: []byte("{ \"key\": \"value\" }"),
					DecryptedSecureJSONData: map[string]string{
						"sKey": "sValue",
					},
				},
				expectedClientOptions: httpclient.Options{
					CustomOptions: map[string]interface{}{
						dataCustomOptionsKey: map[string]interface{}{
							"key": "value",
						},
						secureDataCustomOptionsKey: map[string]string{
							"sKey": "sValue",
						},
					},
				},
			},
		}

		for _, tc := range tcs {
			opts, err := tc.instanceSettings.HTTPClientOptions()
			assert.NoError(t, err)
			if tc.expectedClientOptions.BasicAuth != nil {
				assert.Equal(t, tc.expectedClientOptions.BasicAuth, opts.BasicAuth)
			} else {
				assert.Nil(t, opts.BasicAuth)
			}

			if tc.expectedClientOptions.Labels != nil {
				assert.Equal(t, tc.expectedClientOptions.Labels, opts.Labels)
			}

			jsonData := JSONDataFromHTTPClientOptions(opts)
			expectedJSONData := JSONDataFromHTTPClientOptions(tc.expectedClientOptions)
			secureJSONData := SecureJSONDataFromHTTPClientOptions(opts)
			expectedSecureJSONData := SecureJSONDataFromHTTPClientOptions(tc.expectedClientOptions)

			if len(tc.expectedClientOptions.CustomOptions) > 0 {
				assert.Equal(t, tc.expectedClientOptions.CustomOptions, opts.CustomOptions)
				assert.Equal(t, expectedJSONData, jsonData)
				assert.Equal(t, expectedSecureJSONData, secureJSONData)
			} else {
				assert.Empty(t, opts.CustomOptions)
				assert.Empty(t, jsonData)
				assert.Empty(t, secureJSONData)
			}
		}
	})
}

func TestCustomOptions(t *testing.T) {
	t.Run("Should be able to extract JSONData and SecureJSONData from custom options", func(t *testing.T) {
		opts := &httpclient.Options{}
		expectedJSONData := map[string]interface{}{
			"key": "value",
		}
		expectedSecureJSONData := map[string]string{
			"sKey": "sValue",
		}
		setCustomOptionsFromHTTPSettings(opts, &HTTPSettings{
			JSONData:       expectedJSONData,
			SecureJSONData: expectedSecureJSONData,
		})

		jsonData := JSONDataFromHTTPClientOptions(*opts)
		secureJSONData := SecureJSONDataFromHTTPClientOptions(*opts)

		require.Equal(t, expectedJSONData, jsonData)
		require.Equal(t, expectedSecureJSONData, secureJSONData)
	})

	t.Run("Should be able to extract JSONData and SecureJSONData from custom options", func(t *testing.T) {
		opts := &httpclient.Options{
			CustomOptions: map[string]interface{}{},
		}
		incorrectJSONData := map[string]string{
			"key": "value",
		}
		incorrectSecureJSONData := map[string]interface{}{
			"sKey": "sValue",
		}
		opts.CustomOptions[dataCustomOptionsKey] = incorrectJSONData
		opts.CustomOptions[secureDataCustomOptionsKey] = incorrectSecureJSONData

		jsonData := JSONDataFromHTTPClientOptions(*opts)
		secureJSONData := SecureJSONDataFromHTTPClientOptions(*opts)

		require.Empty(t, jsonData)
		require.Empty(t, secureJSONData)
	})
}
