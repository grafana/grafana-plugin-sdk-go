package query

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type TypedQueryHandler[Q any] interface {
	QueryTypeDefinitionsJSON() (json.RawMessage, error)

	// Get the query parser for a query type
	// The version is split from the end of the discriminator field
	ReadQuery(
		// The query type split by version (when multiple exist)
		queryType string, version string,
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

// K8s placeholder
type ObjectMeta struct {
	Name              string `json:"name,omitempty"`            // missing on lists
	ResourceVersion   string `json:"resourceVersion,omitempty"` // indicates something changed
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
}

type QueryTypeDefinitionSpec struct {
	// The query type value
	// NOTE: this must be a k8s compatible name
	Name string `json:"name,omitempty"` // must be k8s name? compatible

	// DiscriminatorField is the field used to link behavior to this specific
	// query type.  It is typically "queryType", but can be another field if necessary
	DiscriminatorField string `json:"discriminatorField,omitempty"`

	// Describe whe the query type is for
	Description string `json:"description,omitempty"`

	// Versions (most recent first)
	Versions []QueryTypeVersion `json:"versions"`

	// When multiple versions exist, this is the preferredVersion
	PreferredVersion string `json:"preferredVersion,omitempty"`
}

type QueryTypeVersion struct {
	// Version identifier or empty if only one exists
	Version string `json:"version,omitempty"`

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
