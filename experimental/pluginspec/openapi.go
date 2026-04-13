package pluginspec

import (
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/yaml"
)

// Get the OpenAPI info for a given version (eg, v0alpha1 or v1)
type OpenAPIExtensionProvider = func(apiVersion string) (*OpenAPIExtension, error)

// Define the plugin settings and routes in the OpenAPI spec
type OpenAPIExtension struct {
	// Defines the configuration schema for a plugin (datasource or app)
	Settings Settings `json:"settings"`

	// Define the resource and proxy routes
	Routes *Routes `json:"routes,omitempty"`

	// Additional Schemas added to the result, and may be referenced by the routes above
	Schemas map[string]*spec.Schema `json:"schemas,omitempty"`
}

func (o OpenAPIExtension) ToYAML() ([]byte, error) {
	return yaml.Marshal(o) // this is a k8s compatible format
}

func LoadSpec(jsonOrYaml []byte) (*OpenAPIExtension, error) {
	obj := &OpenAPIExtension{}
	err := yaml.Unmarshal(jsonOrYaml, obj)
	return obj, err
}

// Define the configuration object
type Settings struct {
	// Define the spec section of the resource settings configuration
	// jsonData will be a child of this object and the siblings should include any valid options
	// except for secure values -- these are defined by the the `secureValues` property below
	Spec *spec.Schema `json:"spec"`

	// Define which secure values are required
	SecureValues []SecureValueInfo `json:"secureValues,omitempty"`

	// Examples added to the swagger documentation
	Examples map[string]*spec3.Example `json:"examples,omitempty"`
}

type SecureValueInfo struct {
	// The secure value key
	Key string `json:"key"`

	// Description
	Description string `json:"description,omitempty"`

	// Required secure values
	Required bool `json:"required,omitempty"`
}

type Routes struct {
	// Resource routes -- define the paths under "resources":
	// DataSource:
	// - {group}/{version}/namespaces/{ns}/datasource/{name}/resources/{route}
	// Apps:
	// - {group}/{version}/namespaces/{ns}/resources/{route}
	Resource map[string]*spec3.Path `json:"resource,omitempty"`

	// Proxy routes -- the paths exposed under:
	// DataSource:
	// - {group}/{version}/namespaces/{ns}/datasource/{name}/proxy/{route}
	// Apps:
	// - {group}/{version}/namespaces/{ns}/proxy/{route}
	Proxy map[string]*spec3.Path `json:"proxy,omitempty"`
}
