package entity

import "github.com/grafana/grafana-plugin-sdk-go/data"

type IndexField struct {
	// Name is the field name
	Name string `json:"name,omitempty"`

	// IsUnique is a hint that values for this field will be unique across the corpus
	IsUnique bool `json:"unique,omitempty"`

	// Type maps to the DataFrame field type
	// currently JSON will be used for anything multi-valued
	Type data.FieldType `json:"type"`

	// Config is optional display configuration information for Grafana.  This can include units and description
	Config *data.FieldConfig `json:"config,omitempty"`
}
