package pluginspec_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	dsV0 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginspec"
)

func TestNewProviderFromFS_LoadsYAML(t *testing.T) {
	type openapiExpect struct {
		found       bool
		err         string
		description string
	}
	type queryExpect struct {
		found bool
		err   string
	}

	testCases := []struct {
		name    string
		fsys    fs.FS
		version string
		openapi openapiExpect
		query   queryExpect
	}{{
		name:    "not found",
		version: "v0alpha1",
		fsys:    fstest.MapFS{},
	}, {
		name:    "yaml with spec description",
		version: "v0alpha1",
		openapi: openapiExpect{
			found:       true,
			description: "Test",
		},
		fsys: fstest.MapFS{
			"spec.v0alpha1.openapi.yaml": &fstest.MapFile{
				Data: []byte(`settings:
  spec:
    description: Test`),
			},
		},
	}, {
		name:    "json with spec description",
		version: "v0alpha1",
		openapi: openapiExpect{
			found:       true,
			description: "Test",
		},
		fsys: fstest.MapFS{
			"spec.v0alpha1.openapi.json": &fstest.MapFile{
				Data: []byte(`{"settings": {"spec": { "description": "Test"}}}`),
			}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := pluginspec.NewSpecProvider(tc.fsys)

			// Check OpenAPI
			//---------------
			ext, err := provider.GetOpenAPI(tc.version)
			if tc.openapi.err != "" {
				require.ErrorContains(t, err, tc.openapi.err)
			} else {
				require.NoError(t, err)
			}
			if tc.openapi.found {
				require.NotNil(t, ext, "expect something")
			} else {
				require.Nil(t, ext, "expect nothing")
			}
			if tc.openapi.description != "" {
				require.Equal(t, tc.openapi.description, ext.Settings.Spec.Description)
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
