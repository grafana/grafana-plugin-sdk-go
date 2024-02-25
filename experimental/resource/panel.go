package resource

type PseudoPanel[Q any] struct {
	// Numeric panel id
	ID int `json:"id,omitempty"`

	// The panel plugin type
	Type string `json:"type"`

	// The panel title
	Title string `json:"title,omitempty"`

	// Options depend on the panel type
	Options map[string]any `json:"options,omitempty"`

	// FieldConfig values depend on the panel type
	FieldConfig map[string]any `json:"fieldConfig,omitempty"`

	// This should no longer be necessary since each target has the datasource reference
	Datasource *DataSourceRef `json:"datasource,omitempty"`

	// The query targets
	Targets []Q `json:"targets"`
}
