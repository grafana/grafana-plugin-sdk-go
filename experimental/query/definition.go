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
	QueryTypes() []QueryTypeDefinition

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

type QueryTypeDefinitions struct {
	// Describe whe the query type is for
	Field string `json:"field,omitempty"`

	Types []QueryTypeDefinition `json:"types"`
}

type QueryTypeDefinition struct {
	// Describe whe the query type is for
	Name string `json:"name,omitempty"`

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
