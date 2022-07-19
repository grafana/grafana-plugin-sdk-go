package entity

// Everything except body -- so it can be extended
type ConcreteEntityBase struct {
	// Path includes slashes
	Path string `json:"path,omitempty"`
	// dash, ds, alert, folder, svg, png, df, dqr, ... (will validate body)
	Kind string `json:"kind,omitempty"`
	// v1  -- defines the wrapper
	ApiVersion string `json:"apiVersion,omitempty"`
	// defines the body contents
	SchemaVersion string `json:"schemaVersion,omitempty"`
	// common user defined properties avaliable for everything
	Props *EntityProperties `json:"props,omitempty"`
	// Metadata managed by the underlying storage engine(s)
	Meta *StorageMetadata `json:"meta,omitempty"`
}
