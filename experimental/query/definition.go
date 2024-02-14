package query

import (
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type TypedQueryReader[Q any] interface {
	// Get the query parser for a query type
	// The version is split from the end of the discriminator field
	ReadQuery(
		// Properties that have been parsed off the same node
		common CommonQueryProperties,
		// An iterator with context for the full node (include common values)
		iter *jsoniter.Iterator,
	) (Q, error)
}

// K8s placeholder
// This will serialize to the same byte array, but does not require all the imports
type QueryTypeDefinition struct {
	ObjectMeta ObjectMeta `json:"metadata,omitempty"`

	Spec QueryTypeDefinitionSpec `json:"spec,omitempty"`
}

// K8s placeholder
// This will serialize to the same byte array, but does not require all the imports
type QueryTypeDefinitionList struct {
	ObjectMeta ObjectMeta            `json:"metadata,omitempty"`
	Items      []QueryTypeDefinition `json:"items"`
}

// K8s compatible
type ObjectMeta struct {
	// The name is for k8s and description, but not used in the schema
	Name string `json:"name,omitempty"`
	// Changes indicate that *something * changed
	ResourceVersion string `json:"resourceVersion,omitempty"`
	// Timestamp
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
}

type QueryTypeDefinitionSpec struct {
	// DiscriminatorField is the field used to link behavior to this specific
	// query type.  It is typically "queryType", but can be another field if necessary
	DiscriminatorField string `json:"discriminatorField,omitempty"`

	// The discriminator value
	DiscriminatorValue string `json:"discriminatorValue,omitempty"`

	// Describe whe the query type is for
	Description string `json:"description,omitempty"`

	// The JSONSchema definition for the non-common fields
	Schema any `json:"schema"`

	// Examples (include a wrapper) ideally a template!
	Examples []QueryExample `json:"examples,omitempty"`

	// Changelog defines the changed from the previous version
	// All changes in the same version *must* be backwards compatible
	// Only notable changes will be shown here, for the full version history see git!
	Changelog []string `json:"changelog,omitempty"`
}

type QueryExample struct {
	// Version identifier or empty if only one exists
	Name string `json:"name,omitempty"`

	// An example query
	Query any `json:"query"`
}
