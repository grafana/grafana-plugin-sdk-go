package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema"
	"github.com/magefile/mage/mg"
)

// Datasource datasource extraction commands.
type Datasource mg.Namespace

const (
	openAPIFilename    = "spec.v0alpha1.openapi.json"
	queryTypesFilename = "spec.v0alpha1.query.types.json"
)

// GenerateOpenAPI generates datasource OpenAPI JSON and writes spec.v0alpha1.openapi.json in the plugin directory.
func (Datasource) GenerateOpenAPI(dir *string) error {
	pluginDir, err := normalizePluginDir(optionalDir(dir))
	if err != nil {
		return err
	}

	result, err := datasourceschema.GenerateOpenAPI(datasourceschema.OpenAPIOptions{Dir: pluginDir})
	if err != nil {
		return err
	}

	return writeNamedFile(pluginDir, openAPIFilename, result.Body, result.Warnings)
}

// GenerateQueryTypes generates datasource query type JSON and writes spec.v0alpha1.query.types.json in the plugin directory.
func (Datasource) GenerateQueryTypes(dir *string) error {
	pluginDir, err := normalizePluginDir(optionalDir(dir))
	if err != nil {
		return err
	}

	result, err := datasourceschema.GenerateQueryTypes(datasourceschema.OpenAPIOptions{Dir: pluginDir})
	if err != nil {
		return err
	}

	return writeNamedFile(pluginDir, queryTypesFilename, result.Body, result.Warnings)
}

func normalizePluginDir(dir string) (string, error) {
	pluginDir := dir
	if pluginDir == "" {
		pluginDir = "."
	}
	if os.Getenv("GOTOOLCHAIN") == "" {
		if err := os.Setenv("GOTOOLCHAIN", "auto"); err != nil {
			return "", err
		}
	}
	return pluginDir, nil
}

func optionalDir(dir *string) string {
	if dir == nil {
		return ""
	}

	return *dir
}

func writeOpenAPIWarnings(f *os.File, warnings []datasourceschema.OpenAPIWarning) error {
	for _, warning := range warnings {
		if _, err := fmt.Fprintf(f, "warning: %s: %s", warning.Code, warning.Message); err != nil {
			return err
		}
		if warning.Position.File != "" {
			if _, err := fmt.Fprintf(f, " (%s:%d:%d)", warning.Position.File, warning.Position.Line, warning.Position.Column); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(f); err != nil {
			return err
		}
	}

	return nil
}

func writeNamedFile(dir string, name string, body []byte, warnings []datasourceschema.OpenAPIWarning) error {
	if err := writeOpenAPIWarnings(os.Stderr, warnings); err != nil {
		return err
	}

	path := filepath.Join(dir, name)
	return os.WriteFile(path, append([]byte{}, body...), 0o600)
}
