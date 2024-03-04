package v0alpha1

type PseudoPanel struct {
	// Numeric panel id
	ID int `json:"id,omitempty"`

	// The panel plugin type
	Type string `json:"type"`

	// The panel title
	Title string `json:"title,omitempty"`

	// This should no longer be necessary since each target has the datasource reference
	Datasource *DataSourceRef `json:"datasource,omitempty"`

	// The query targets
	Targets []DataQuery `json:"targets"`
}
