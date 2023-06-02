package hash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestHashDataSourceSettings(t *testing.T) {
	tcs := []struct {
		name         string
		settings     backend.DataSourceInstanceSettings
		expectedHash string
	}{
		{
			name:         "Zeroed value",
			settings:     backend.DataSourceInstanceSettings{},
			expectedHash: "21367070002710377",
		},
		{
			name: "Simple",
			settings: backend.DataSourceInstanceSettings{
				ID:  1,
				UID: "123abc",
				URL: "http://localhost:9090",
			},
			expectedHash: "9111523014157213054",
		},
		{
			name: "Simple (modified)",
			settings: backend.DataSourceInstanceSettings{
				ID:  1,
				UID: "123abc",
				URL: "http://localhost:9090/",
			},
			expectedHash: "3940950478756612732",
		},
		{
			name: "Update field is not considered",
			settings: backend.DataSourceInstanceSettings{
				ID:      1,
				UID:     "123abc",
				URL:     "http://localhost:9090/",
				Updated: time.Now(),
			},
			expectedHash: "3940950478756612732",
		},
		{
			name: "Complex",
			settings: backend.DataSourceInstanceSettings{
				ID:                      1,
				UID:                     "123abc",
				Type:                    "test-datasource",
				Name:                    "Test",
				URL:                     "http://localhost:9090",
				User:                    "Test",
				JSONData:                []byte(`{"foo": "bar"}`),
				DecryptedSecureJSONData: map[string]string{"token": "secret"},
			},
			expectedHash: "1137523268333599661",
		},
		{
			name: "Complex (modified)",
			settings: backend.DataSourceInstanceSettings{
				ID:                      1,
				UID:                     "123abc",
				Type:                    "test-datasource",
				Name:                    "Test",
				URL:                     "http://localhost:9090",
				User:                    "Test",
				JSONData:                []byte(`{"foo": "baz"}`),
				DecryptedSecureJSONData: map[string]string{"token": "secret"},
			},
			expectedHash: "10853666091215604292",
		},
		{
			name: "Complex (modified again)",
			settings: backend.DataSourceInstanceSettings{
				ID:   1,
				UID:  "123abc",
				Type: "test-datasource",
				Name: "Test",
				URL:  "http://localhost:9090",
				User: "Test",
				DecryptedSecureJSONData: map[string]string{
					"foo":   "baz",
					"token": "secret",
				},
			},
			expectedHash: "8741997558572900423",
		},
		{
			name: "Invalid JSON",
			settings: backend.DataSourceInstanceSettings{
				JSONData: []byte(`does not matter`),
			},
			expectedHash: "3741731363373401139",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := DataSourceSettings(tc.settings)
			require.NoError(t, err)
			require.Equal(t, tc.expectedHash, hash)
		})
	}
}
