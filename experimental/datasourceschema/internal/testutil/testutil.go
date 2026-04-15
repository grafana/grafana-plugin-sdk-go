package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testDirPerm  = 0o750
	testFilePerm = 0o600
)

func WriteFixtureModule(tb testing.TB, files map[string]string) string {
	tb.Helper()

	dir := tb.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(dir, name)
		require.NoError(tb, os.MkdirAll(filepath.Dir(fullPath), testDirPerm), "mkdir failed for %s", fullPath)
		require.NoError(tb, os.WriteFile(fullPath, []byte(strings.TrimLeft(content, "\n")), testFilePerm), "write failed for %s", fullPath)
	}

	return dir
}

func NestedMap(value map[string]any, keys ...string) (map[string]any, bool) {
	current := value
	for _, key := range keys {
		next, ok := current[key]
		if !ok {
			return nil, false
		}

		current, ok = next.(map[string]any)
		if !ok {
			return nil, false
		}
	}

	return current, true
}

func PositionOfSnippet(tb testing.TB, path string, snippet string) (int, int) {
	tb.Helper()

	body, err := os.ReadFile(path) //nolint:gosec // test helper reads from fixture files created during the test.
	require.NoError(tb, err, "read failed: %s", path)

	lines := strings.Split(string(body), "\n")
	for lineIndex, line := range lines {
		column := strings.Index(line, snippet)
		if column >= 0 {
			return lineIndex + 1, column + 1
		}
	}

	require.Failf(tb, "snippet not found", "snippet %q not found in %s", snippet, path)
	return 0, 0
}

func KeysOfMap[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
