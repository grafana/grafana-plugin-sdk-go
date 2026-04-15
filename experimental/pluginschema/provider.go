package pluginschema

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	dsV0 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

type SchemaProvider interface {
	// Valid for both DataSources and Apps
	GetSettings(apiVersion string) (*Settings, error)

	// This will be added as examples for settings
	GetSettingsExamples(apiVersion string) (*SettingsExamples, error)

	// Valid for both DataSources and Apps
	GetRoutes(apiVersion string) (*Routes, error)

	// Only valid for datasources
	// NOTE: this requires passing in the QueryTypeList because the real runtime value
	// The items type MUST be QueryTypeDefinitionSpec
	GetQueryTypes(apiVersion string, queryTypes any) (bool, error)
}

func NewSchemaProvider(fsys fs.FS, prefix string) SchemaProvider {
	return &fsSpecProvider{fs: fsys, prefix: prefix}
}

type fsSpecProvider struct {
	prefix string
	fs     fs.FS
}

// GetOpenAPI implements [SpecProvider].
func (p *fsSpecProvider) GetSettings(apiVersion string) (*Settings, error) {
	raw, err := p.getYAMLorJSON(apiVersion, "settings")
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	s := &Settings{}
	err = Load(raw, s)
	return s, err
}

// GetOpenAPI implements [SpecProvider].
func (p *fsSpecProvider) GetSettingsExamples(apiVersion string) (*SettingsExamples, error) {
	raw, err := p.getYAMLorJSON(apiVersion, "settings.examples")
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	s := &SettingsExamples{}
	err = Load(raw, s)
	return s, err
}

// GetOpenAPI implements [SpecProvider].
func (p *fsSpecProvider) GetRoutes(apiVersion string) (*Routes, error) {
	raw, err := p.getYAMLorJSON(apiVersion, "routes")
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	v := &Routes{}
	err = Load(raw, &v)
	return v, err
}

// GetQueryTypes implements [SpecProvider].
// The queryTypes object is dynamic because the exposed version does not exist in this SDK
func (p *fsSpecProvider) GetQueryTypes(apiVersion string, queryTypes any) (bool, error) {
	raw, err := p.getYAMLorJSON(apiVersion, "query.types")
	if err != nil || len(raw) == 0 {
		return false, err
	}
	err = Load(raw, queryTypes)
	if err != nil {
		return false, err
	}

	// Attach any examples to the requested type
	raw, err = p.getYAMLorJSON(apiVersion, "query.examples")
	if err != nil {
		return false, err
	}
	if len(raw) > 0 {
		examples := &QueryExamples{}
		if err = Load(raw, examples); err != nil {
			return false, err
		}

		// HACK -- for now we will explicitly convert
		types := &dsV0.QueryTypeDefinitionList{}
		if err = Load(raw, queryTypes); err != nil {
			return false, fmt.Errorf("unable to read as QueryTypeDefinitionList %w", err)
		}
		lookup := make(map[string]int, len(types.Items))
		for idx, v := range types.Items {
			lookup[v.Name] = idx
		}
		for _, example := range examples.Examples {
			idx, ok := lookup[example.QueryType]
			if !ok {
				return false, fmt.Errorf("unknown query type")
			}
			types.Items[idx].Spec.Examples = append(types.Items[idx].Spec.Examples, example.Example)
		}

		// replace the raw value with one that has examples
		raw, err = ToYAML(types)
		if err == nil {
			err = Load(raw, queryTypes)
		}
	}

	return true, err
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
