package schema

// ObjectMeta is a struct that aims to "look" like a real kubernetes object when
// written to JSON, however it does not require the pile of dependencies
// This is really an internal helper until we decide which dependencies make sense
// to require within the SDK
type ObjectMeta struct {
	// The name is for k8s and description, but not used in the schema
	Name string `json:"name,omitempty"`
	// Changes indicate that *something * changed
	ResourceVersion string `json:"resourceVersion,omitempty"`
	// Timestamp
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
}

// QueryTypeDefinition is a kubernetes shaped object that represents a single query definition
type QueryTypeDefinition struct {
	ObjectMeta ObjectMeta `json:"metadata,omitempty"`

	Spec QueryTypeDefinitionSpec `json:"spec,omitempty"`
}

// QueryTypeDefinitionList is a kubernetes shaped object that represents a list of query types
// For simple data sources, there may be only a single query type, however when multiple types
// exist they must be clearly specified with distinct discriminator field+value pairs
type QueryTypeDefinitionList struct {
	Kind       string `json:"kind"`       // "QueryTypeDefinitionList",
	ApiVersion string `json:"apiVersion"` // "query.grafana.app/v0alpha1",

	ObjectMeta `json:"metadata,omitempty"`

	Items []QueryTypeDefinition `json:"items"`
}

// SettingsDefinition is a kubernetes shaped object that represents a single query definition
type SettingsDefinition struct {
	ObjectMeta ObjectMeta `json:"metadata,omitempty"`

	Spec SettingsDefinitionSpec `json:"spec,omitempty"`
}

// QueryTypeDefinitionList is a kubernetes shaped object that represents a list of query types
// For simple data sources, there may be only a single query type, however when multiple types
// exist they must be clearly specified with distinct discriminator field+value pairs
type SettingsDefinitionList struct {
	Kind       string `json:"kind"`       // "SettingsDefinitionList",
	ApiVersion string `json:"apiVersion"` // "??.common.grafana.app/v0alpha1",

	ObjectMeta `json:"metadata,omitempty"`

	Items []SettingsDefinition `json:"items"`
}
