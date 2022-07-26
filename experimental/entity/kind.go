package entity

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type KindRegistry interface {
	Get(id string) Kind
	GetFromSuffix(path string) Kind
	List() []Kind
	Register(kinds ...Kind) error
}

type KindInfo struct {
	ID          string `json:"ID,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	// Detect the kind from a path suffix
	PathSuffix string `json:"pathSuffix,omitempty"`

	// For kinds with secure keys -- the keys will be strpped unless user has editor access
	HasSecureKeys bool `json:"hasSecureKeys,omitempty"`

	// The entity store does not extend the base EntityEnvelope -- this is typical for
	// non-object-model formats like images (png, svg, etc)
	IsRaw bool `json:"isRaw,omitempty"`

	// For raw content types, this is set as an HTTP header
	ContentType string `json:"contentType,omitempty"`
}

type Kind interface {
	Info() KindInfo

	// Called before saving any object.  The result will be sanitized and safe to write on disk
	Normalize(payload []byte, details bool) NormalizeResponse

	// Modify the object payload
	Migrate(payload []byte, targetVersion string) NormalizeResponse

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

type NormalizeResponse struct {
	Valid bool `json:"valid"`

	// This includes potential errors and warnings
	Info []data.Notice `json:"info,omitempty"`

	// Some kinds may have more detailed response objects
	Details interface{} `json:"details,omitempty"`

	// When this exists, the payload has been sanitized and is considered safe to save
	Result []byte `json:"result,omitempty"`
}

// The Kind indicator for folders
const FolderKindID = "folder"

// FolderKind is the kind used to indicate a folder.  Folders do not have a body, but can include a listing
var FolderKind = NewGenericKind(KindInfo{
	ID:          FolderKindID,
	Description: "Folder",
})
