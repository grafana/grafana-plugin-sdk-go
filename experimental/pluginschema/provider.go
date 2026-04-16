package pluginschema

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
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
