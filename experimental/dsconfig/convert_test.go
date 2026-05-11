package dsconfig_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/dsconfig"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToPluginSettings_Table runs file-based table tests.
// Each subdirectory under testdata/convert/ contains:
//   - input.json:  DatasourceConfigSchema
//   - output.json: expected pluginschema.Settings
//   - config.json: example Grafana storage model (documentation only, not asserted)
func TestToPluginSettings_Table(t *testing.T) {
	testdataDir := filepath.Join("testdata", "convert")

	entries, err := os.ReadDir(testdataDir)
	require.NoError(t, err, "failed to read testdata/convert directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dir := filepath.Join(testdataDir, name)

		if _, err := os.Stat(filepath.Join(dir, "input.json")); os.IsNotExist(err) {
			continue
		}

		t.Run(name, func(t *testing.T) {
			inputBytes, err := os.ReadFile(filepath.Join(dir, "input.json")) //nolint:gosec // test fixture
			require.NoError(t, err)

			expectedBytes, err := os.ReadFile(filepath.Join(dir, "output.json")) //nolint:gosec // test fixture
			require.NoError(t, err)

			var inputSchema dsconfig.DatasourceConfigSchema
			require.NoError(t, json.Unmarshal(inputBytes, &inputSchema))

			got, err := inputSchema.ToPluginSettings()
			require.NoError(t, err, "ToPluginSettings() returned error")

			var expected pluginschema.Settings
			require.NoError(t, json.Unmarshal(expectedBytes, &expected))

			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			expectedJSON, err := json.Marshal(expected)
			require.NoError(t, err)

			var gotMap, expectedMap map[string]any
			require.NoError(t, json.Unmarshal(gotJSON, &gotMap))
			require.NoError(t, json.Unmarshal(expectedJSON, &expectedMap))

			assert.Equal(t, expectedMap, gotMap)
		})
	}
}
