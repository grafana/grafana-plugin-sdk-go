package v0alpha1

// SettingsDefinition is a kubernetes shaped object that represents a single query definition
type SettingsDefinition struct {
	ObjectMeta ObjectMeta `json:"metadata,omitempty"`

	Spec SettingsDefinitionSpec `json:"spec,omitempty"`
}

// QueryTypeDefinitionList is a kubernetes shaped object that represents a list of query types
// For simple data sources, there may be only a single query type, however when multiple types
// exist they must be clearly specified with distinct discriminator field+value pairs
type SettingsDefinitionList struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	Items []SettingsDefinition `json:"items"`
}

type SettingsDefinitionSpec struct {
	// Multiple schemas can be defined using discriminators
	Discriminators []DiscriminatorFieldValue `json:"discriminators,omitempty"`

	// Describe whe the query type is for
	Description string `json:"description,omitempty"`

	// The query schema represents the properties that can be sent to the API
	// In many cases, this may be the same properties that are saved in a dashboard
	// In the case where the save model is different, we must also specify a save model
	JSONDataSchema JSONSchema `json:"jsonDataSchema"`

	// JSON schema defining the properties needed in secure json
	// NOTE all properties must be string values!
	SecureProperties JSONSchema `json:"secureJsonSchema"`

	// Changelog defines the changed from the previous version
	// All changes in the same version *must* be backwards compatible
	// Only notable changes will be shown here, for the full version history see git!
	Changelog []string `json:"changelog,omitempty"`
}
