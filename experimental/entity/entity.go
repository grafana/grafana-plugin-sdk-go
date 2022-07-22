package entity

// This is the base class for (non-raw) kinds
type Envelope struct {
	// UID may contain / to indicate parent folder's
	UID string `json:"UID,omitempty"`

	// dash, ds, alert, folder, svg, png, df, dqr, ... (will validate body)
	Kind string `json:"kind,omitempty"`

	// Version of the included body (semver)
	SchemaVersion string `json:"schemaVersion,omitempty"`

	// Entity name
	Name string `json:"name,omitempty"`

	// Entity description
	Description string `json:"description,omitempty"`

	// Tags to add for search
	Labels map[string]string `json:"labels,omitempty"`

	// Optional metadata describing where the body came from
	Provinance *Provinance `json:"provinance,omitempty"`
}

// Extension to the core entity wrapper that supports managed secure keys
type EnvelopeWithSecureKeys struct {
	Envelope

	// NOTE: Although APIs will limit exposing this to people without permissions,
	// The contents of this map (key+value) may be stored in externally secured object stores (S3, disk, etc)
	// The values should be lookup keys into a secret service
	SecureKeys map[string]string `json:"secureKeys,omitempty"`
}

// Define how an item got into the system
type Provinance struct {
	// Unix millis when the event happened
	When int64 `json:"when,omitempty"`
	// Identifier for the source.  ex: provisioning
	Source string `json:"source,omitempty"`
	// optional path to the original source
	Path string `json:"path,omitempty"`
}

type EntityLocator struct {
	// UID may contain / to indicate parent folder's
	UID string `json:"UID,omitempty"`

	// dasboard, datasource, etc
	Kind string `json:"kind,omitempty"`

	// prometheus etc
	Type string `json:"type,omitempty"`
}
