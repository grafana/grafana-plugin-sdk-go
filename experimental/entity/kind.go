package entity

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type Kind interface {
	Info() *KindInfo

	// Called before saving any object.  The result will be sanitized and safe to write on disk
	Sanitize(payload []byte, details bool) ValidationResponse

	// Modify the object payload
	Migrate(payload []byte, targetVersion string) ValidationResponse

	// Marshal raw payload into an entity type.  The resulting interface will implement `Envelope`
	Read(payload []byte) (interface{}, error)

	// Given a well defined object, create the expected payload
	Write(interface{}) ([]byte, error)

	// Identify referenced items
	GetReferences(interface{}) []EntityLocator

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
