package pluginspec

import (
	"errors"
	"io/fs"
	"strings"

	"sigs.k8s.io/yaml"
)

type SpecProvider interface {
	// Valid for both DataSources and Apps
	GetOpenAPI(apiVersion string) (*OpenAPIExtension, error)

	// Only valid for datasources
	GetQueryTypes(apiVersion string, queryTypes any) (bool, error)
}

func NewSpecProvider(fsys fs.FS) SpecProvider {
	return &fsSpecProvider{fsys}
}

type fsSpecProvider struct {
	fsys fs.FS
}

// GetOpenAPI implements [SpecProvider].
func (p *fsSpecProvider) GetOpenAPI(apiVersion string) (*OpenAPIExtension, error) {
	raw, err := p.getYAMLorJSON("spec." + apiVersion + ".openapi")
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return LoadSpec(raw)
}

// GetQueryTypes implements [SpecProvider].
// The queryTypes object is dynamic because the exposed version does not exist in this SDK
func (p *fsSpecProvider) GetQueryTypes(apiVersion string, queryTypes any) (bool, error) {
	raw, err := p.getYAMLorJSON("spec." + apiVersion + ".query.types")
	if err != nil || len(raw) == 0 {
		return false, err
	}
	err = yaml.Unmarshal(raw, queryTypes)
	return true, err
}

func (p *fsSpecProvider) getYAMLorJSON(prefix string) ([]byte, error) {
	data, err := fs.ReadFile(p.fsys, prefix+".yaml")
	if isNotExists(err) {
		data, err = fs.ReadFile(p.fsys, prefix+".json")
		if isNotExists(err) {
			return nil, nil // does not exist
		}
	}
	return data, err
}

func isNotExists(err error) bool {
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return strings.Contains(err.Error(), "file does not exist") // from the os filesystem :(
}
