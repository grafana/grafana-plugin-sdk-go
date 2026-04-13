package pluginspec_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginspec"
)

func TestNewProviderFromFS_LoadsYAML(t *testing.T) {
	testCases := []struct {
		name          string
		fsys          fs.FS
		version       string
		expectErr     string
		expectedTitle string
	}{{
		name:    "not found",
		version: "v0alpha1",
		fsys:    fstest.MapFS{},
	}, {
		name:          "yaml with spec description",
		version:       "v0alpha1",
		expectedTitle: "Test",
		fsys: fstest.MapFS{
			"spec.v0alpha1.openapi.yaml": &fstest.MapFile{
				Data: []byte(`settings:
  spec:
    description: Test`),
			},
		},
	}, {
		name:          "json with spec description",
		version:       "v0alpha1",
		expectedTitle: "Test",
		fsys: fstest.MapFS{
			"spec.v0alpha1.openapi.json": &fstest.MapFile{
				Data: []byte(`{"settings": {"spec": { "description": "Test"}}}`),
			}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := pluginspec.NewSpecProvider(tc.fsys)

			ext, err := provider.GetOpenAPI(tc.version)
			if tc.expectErr != "" {
				require.ErrorContains(t, err, tc.expectErr)
			} else {
				if tc.expectedTitle == "" {
					require.Nil(t, ext) // not found
				} else {
					require.NoError(t, err)
					require.NotNil(t, ext)
					require.Equal(t, tc.expectedTitle, ext.Settings.Spec.Description)
				}
			}
		})
	}
}
