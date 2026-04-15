package pluginschema_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	dsV0 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

func TestNewProviderFromFS_LoadsYAML(t *testing.T) {
	type settingsExpect struct {
		found       bool
		err         string
		description string
	}
	type queryExpect struct {
		found bool
		err   string
	}

	testCases := []struct {
		name     string
		fs       fs.FS
		version  string
		settings settingsExpect
		query    queryExpect
	}{{
		name:    "not found",
		version: "v0alpha1",
		fs:      fstest.MapFS{},
	}, {
		name:    "yaml with spec description",
		version: "v0alpha1",
		settings: settingsExpect{
			found:       true,
			description: "Test",
		},
		fs: fstest.MapFS{
			"v0alpha1/settings.yaml": &fstest.MapFile{
				Data: []byte(`spec:
    description: Test`),
			},
		},
	}, {
		name:    "json with spec description",
		version: "v0alpha1",
		settings: settingsExpect{
			found:       true,
			description: "Test",
		},
		fs: fstest.MapFS{
			"v0alpha1/settings.json": &fstest.MapFile{
				Data: []byte(`{"spec": { "description": "Test"}}`),
			}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := pluginschema.NewSchemaProvider(tc.fs, "")

			// Check OpenAPI
			//---------------
			ext, err := provider.GetSettings(tc.version)
			if tc.settings.err != "" {
				require.ErrorContains(t, err, tc.settings.err)
			} else {
				require.NoError(t, err)
			}
			if tc.settings.found {
				require.NotNil(t, ext, "expect something")
			} else {
				require.Nil(t, ext, "expect nothing")
			}
			if tc.settings.description != "" {
				require.Equal(t, tc.settings.description, ext.Spec.Description)
			}

			// Check QueryTypes
			//-----------------
			qt := &dsV0.QueryTypeDefinitionList{}
			found, err := provider.GetQueryTypes(tc.version, qt)
			if tc.query.err != "" {
				require.ErrorContains(t, err, tc.query.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.query.found, found)
		})
	}
}
