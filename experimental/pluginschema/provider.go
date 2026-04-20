package pluginschema

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"sigs.k8s.io/yaml"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

type SchemaProvider interface {
	Get(apiVersion string) (*PluginSchema, error)
}

type PluginSchema struct {
	// The apiVersion where this schema applies
	APIVersion string

	// Defines the settings (configuration) object
	SettingsSchema *Settings

	// Explore example settings
	SettingsExamples *SettingsExamples

	// Defines the OpenAPI routes (and additional components)
	// Supports: /resources/*, and /proxy/*
	Routes *Routes

	// Define schemas for different query types
	// NOTE, this is only valid for DataSource plugins
	QueryTypes *sdkapi.QueryTypeDefinitionList

	// A list of example queries
	// NOTE, this is only valid for DataSource plugins
	QueryExamples *sdkapi.QueryExamples
}

func (s *PluginSchema) IsZero() bool {
	if s == nil {
		return true
	}
	if s.SettingsSchema != nil && !s.SettingsSchema.IsZero() {
		return false
	}
	if s.SettingsExamples != nil && !s.SettingsExamples.IsZero() {
		return false
	}
	if s.Routes != nil && !s.Routes.IsZero() {
		return false
	}
	if s.QueryTypes != nil && len(s.QueryTypes.Items) > 0 {
		return false
	}
	if s.QueryExamples != nil && len(s.QueryExamples.Examples) > 0 {
		return false
	}
	return true
}

// Loads a PluginSchema from read-only files.  Specifically:
// - {apiVersion}/settings.{yaml|json}
// - {apiVersion}/settings.examples.{yaml|json}
// - {apiVersion}/routes.{yaml|json}
// - {apiVersion}/query.types.{yaml|json}
// - {apiVersion}/query.examples.{yaml|json}
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
	exists := false
	schema := &PluginSchema{APIVersion: apiVersion}

	// Settings
	raw, err := p.getYAMLorJSON(apiVersion, "settings")
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		exists = true
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
		exists = true
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
		exists = true
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
		exists = true
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
		exists = true
		schema.QueryExamples = &sdkapi.QueryExamples{}
		if err = Load(raw, schema.QueryExamples); err != nil {
			return nil, err
		}
	}

	if !exists {
		return nil, nil // nothing found!
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

// Load yaml or json into a settings object
func Load(jsonOrYaml []byte, obj any) error {
	return yaml.Unmarshal(jsonOrYaml, obj)
}

// Write settings objects as yaml (k8s compatible flavor)
func ToYAML(obj any) ([]byte, error) {
	return yaml.Marshal(obj) // ensure a k8s compatible format
}
