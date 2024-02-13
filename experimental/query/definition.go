package query

import (
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

type TypedQueryHandler[Q any] interface {
	// QueryTypeField is typically "queryType", but may use a different field to
	// discriminate different field types.  When multiple versions for a field exist,
	// The version identifier is appended to the queryType value after a slash.
	// eg: queryType=showLabels/v2
	QueryTypeField() string

	// Possible query types (if any)
	QueryTypeDefinitions() []QueryTypeDefinitionSpec

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
	Examples []any `json:"examples,omitempty"`

	// Changelog defines the changed from the previous version
	// All changes in the same version *must* be backwards compatible
	// Only notable changes will be shown here, for the full version history see git!
	Changelog []string `json:"changelog,omitempty"`
}
