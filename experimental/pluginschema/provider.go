package pluginschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

type PluginSchema struct {
	APIVersion string

	SettingsSchema   *Settings
	SettingsExamples *SettingsExamples

	// OpenAPI routes
	Routes *Routes

	QueryTypes    *sdkapi.QueryTypeDefinitionList
	QueryExamples *sdkapi.QueryExamples
}

type SchemaProvider interface {
	Get(apiVersion string) (*PluginSchema, error)
}

func NewSchemaProvider(fss fs.FS, prefix string) SchemaProvider {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		panic("the prefix must be a folder path ending with /")
	}
	return &fsSpecProvider{fs: fss, prefix: prefix}
}

type fsSpecProvider struct {
	prefix string
	fs     fs.FS
}

func (p *fsSpecProvider) Get(apiVersion string) (*PluginSchema, error) {
	schema := &PluginSchema{APIVersion: apiVersion}

	// Settings
	raw, err := p.getYAMLorJSON(apiVersion, "settings")
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		schema.SettingsSchema = &Settings{}
		if err = Load(raw, schema.SettingsSchema); err != nil {
			return nil, err
		}
	}

	// SettingsExamples
	raw, err = p.getYAMLorJSON(apiVersion, "settings.examples")
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		schema.SettingsExamples = &SettingsExamples{}
		if err = Load(raw, schema.SettingsExamples); err != nil {
			return nil, err
		}
	}

	// Routes
	raw, err = p.getYAMLorJSON(apiVersion, "routes")
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		schema.Routes = &Routes{}
		if err = Load(raw, schema.Routes); err != nil {
			return nil, err
		}
	}

	// QueryTypes
	raw, err = p.getYAMLorJSON(apiVersion, "query.types")
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		schema.QueryTypes = &sdkapi.QueryTypeDefinitionList{}
		if err = Load(raw, schema.QueryTypes); err != nil {
			return nil, err
		}
	}

	// QueryExamples
	raw, err = p.getYAMLorJSON(apiVersion, "query.examples")
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		schema.QueryExamples = &sdkapi.QueryExamples{}
		if err = Load(raw, schema.QueryExamples); err != nil {
			return nil, err
		}
	}

	return schema, nil
}

func (p *fsSpecProvider) getYAMLorJSON(apiVersion, name string) ([]byte, error) {
	path := fmt.Sprintf("%s%s/%s", p.prefix, apiVersion, name)
	data, err := fs.ReadFile(p.fs, path+".json")
	if isNotExists(err) {
		data, err = fs.ReadFile(p.fs, path+".yaml")
		if isNotExists(err) {
			return nil, nil // does not exist
		}
	}
	return data, err
}

func isNotExists(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	// Plugins file system uses string:
	// https://github.com/grafana/grafana/blob/v12.4.2/pkg/plugins/plugins.go#L25
	return strings.Contains(err.Error(), "file does not exist")
}

// Update the schema, failing tests if there are any changes
func UpdateSchema(t *testing.T, s *PluginSchema, dir string) {
	t.Helper()

	require.NotEmpty(t, s.APIVersion)
	provider := NewSchemaProvider(os.DirFS(dir), "")
	current, err := provider.Get(s.APIVersion)
	require.NoError(t, err)
	if current == nil {
		current = &PluginSchema{APIVersion: s.APIVersion}
	}

	write := func(out []byte, name string) {
		fpath := path.Join(dir, s.APIVersion, name)
		err := os.MkdirAll(filepath.Dir(fpath), 0750)
		require.NoError(t, err)
		err = os.WriteFile(fpath, out, 0600)
		require.NoError(t, err)
	}

	if s.SettingsSchema != nil {
		if diff := Diff(s.SettingsSchema, current.SettingsSchema); diff != "" {
			t.Errorf("settings changed (-want +got):\n%s", diff)
			out, err := json.MarshalIndent(s.SettingsSchema, "", "  ")
			require.NoError(t, err)
			write(out, "settings.json")
		}
	}

	if s.SettingsExamples != nil {
		if diff := Diff(s.SettingsExamples, current.SettingsExamples); diff != "" {
			t.Errorf("settings examples changed (-want +got):\n%s", diff)
			out, err := json.MarshalIndent(s.SettingsExamples, "", "  ")
			require.NoError(t, err)
			write(out, "settings.examples.json")
		}
	}

	if s.Routes != nil {
		if diff := Diff(s.Routes, current.Routes); diff != "" {
			t.Errorf("routes changed (-want +got):\n%s", diff)
			out, err := ToYAML(s.Routes)
			require.NoError(t, err)
			write(out, "routes.yaml")
		}
	}

	if s.QueryTypes != nil {
		if diff := Diff(s.QueryTypes, current.QueryTypes); diff != "" {
			t.Errorf("query types changed (-want +got):\n%s", diff)
			out, err := json.MarshalIndent(s.QueryTypes, "", "  ")
			require.NoError(t, err)
			write(out, "query.types.json")
		}
	}

	if s.QueryExamples != nil {
		if diff := Diff(s.QueryExamples, current.QueryExamples); diff != "" {
			t.Errorf("query examples changed (-want +got):\n%s", diff)
			out, err := json.MarshalIndent(s.QueryExamples, "", "  ")
			require.NoError(t, err)
			write(out, "query.examples.json")
		}
	}
}
