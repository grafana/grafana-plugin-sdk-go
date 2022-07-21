package entity

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// User defined properties
type KindInfo struct {
	ID                  string `json:"id"`
	Description         string `json:"description,omitempty"`
	Category            string `json:"category,omitempty"`
	FileSuffix          string `json:"suffix"` // "-dash.json"
	LatestSchemaVersion string `json:"latestSchemaVersion,omitempty"`

	// For kinds with secure keys -- the keys will be strpped unless user has editor access
	HasSecureKeys bool `json:"hasSecureKeys,omitempty"`

	// The entity store does not extend the base EntityEnvelope -- this is typical for
	// non-object-model formats like images (png, svg, etc)
	IsRaw bool `json:"isRaw,omitempty"`
}

type Kind interface {
	Info() KindInfo

	// Called before saving any object.  The result will be sanitized and safe to write on disk
	Validate(payload []byte, details bool) ValidationResponse

	// Modify the object payload
	Migrate(payload []byte, targetVersion string) ValidationResponse

	// Marshal raw payload into an entity type.
	Read(payload []byte) (interface{}, error)

	// Given a well defined object, create the expected payload
	Write(interface{}) ([]byte, error)

	// The expected go type from read
	GoType() interface{}

	// List possible schema versions
	GetSchemaVersions() []string

	// For non-raw formats, this can be used to validate externally
	GetJSONSchema(schemaVersion string) []byte
}

type ValidationResponse struct {
	Valid bool `json:"valid"`

	// This includes potential errors and warnings
	Info []data.Notice `json:"info,omitempty"`

	// Some kinds may have more detailed response objects
	Details interface{} `json:"details,omitempty"`

	// When this exists, the payload has been sanitized and is considered safe to save
	Result []byte `json:"result,omitempty"`
}
