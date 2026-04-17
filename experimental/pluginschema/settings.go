package pluginschema

import (
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// Define the instance settings object
type Settings struct {
	// Define the spec section of the resource settings configuration
	// jsonData will be a child of this object and the siblings should include any valid options
	// except for secure values -- these are defined by the the `secureValues` property below
	Spec *spec.Schema `json:"spec"`

	// Define which secure values are required
	SecureValues []SecureValueInfo `json:"secureValues,omitempty"`
}

func (s Settings) IsZero() bool {
	if s.Spec != nil {
		return false
	}
	if len(s.SecureValues) > 0 {
		return false
	}
	return true
}

type SettingsExamples struct {
	// Example configuration added to the swagger documentation
	Examples map[string]*spec3.Example `json:"examples"`
}

func (s SettingsExamples) IsZero() bool {
	return len(s.Examples) < 1
}

type SecureValueInfo struct {
	// The secure value key
	Key string `json:"key"`

	// Description
	Description string `json:"description,omitempty"`

	// Required secure values
	Required bool `json:"required,omitempty"`
}
