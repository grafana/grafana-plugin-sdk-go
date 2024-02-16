package schema

type SettingsDefinitionSpec struct {
	// DiscriminatorField is the field used to link behavior to this specific
	// query type.  It is typically "queryType", but can be another field if necessary
	DiscriminatorField string `json:"discriminatorField,omitempty"`

	// The discriminator value
	DiscriminatorValue string `json:"discriminatorValue,omitempty"`

	// Describe whe the query type is for
	Description string `json:"description,omitempty"`

	// The query schema represents the properties that can be sent to the API
	// In many cases, this may be the same properties that are saved in a dashboard
	// In the case where the save model is different, we must also specify a save model
	JSONDataSchema any `json:"jsonDataSchema"`

	// JSON schema defining the properties needed in secure json
	// NOTE these must all be string fields
	SecureJSONSchema any `json:"secureJsonSchema"`

	// Changelog defines the changed from the previous version
	// All changes in the same version *must* be backwards compatible
	// Only notable changes will be shown here, for the full version history see git!
	Changelog []string `json:"changelog,omitempty"`
}
