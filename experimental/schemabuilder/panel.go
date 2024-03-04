package schemabuilder

import "github.com/grafana/grafana-plugin-sdk-go/v0alpha1"

// This is only used to write out a sample panel query
// It is not public and not intended to represent a real panel
type pseudoPanel struct {
	// Numeric panel id
	ID int `json:"id,omitempty"`

	// The panel plugin type
	Type string `json:"type"`

	// The panel title
	Title string `json:"title,omitempty"`

	// This should no longer be necessary since each target has the datasource reference
	Datasource *v0alpha1.DataSourceRef `json:"datasource,omitempty"`

	// The query targets
	Targets []v0alpha1.DataQuery `json:"targets"`
}
