package entity

import (
	context "context"

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

	// The expected go type from read
	GoType() interface{}

	// GetIndexFields returns a list of fields that we will be extracted from a payload when indexed
	// The field names will match values returned from the "prepare" call, and can be added to a DataFrame summary
	GetIndexFields() []IndexField

	// The result will be sanitized, safe to write to disk and have clearly identified index data
	Parse(ctx context.Context, payload []byte, details bool) (ParseResponse, error)

	// The result will be sanitized, safe to write to disk and have clearly identified index data
	GetIndexInfo(ctx context.Context, entity interface{}) (IndexResponse, error)

	// Given a go object (m), create the expected payload
	ToPayload(ctx context.Context, entity interface{}) (ParseResponse, error)

	// Migrate from one version to another
	Migrate(ctx context.Context, payload []byte, targetVersion string) ([]byte, error)

	// List possible schema versions
	GetSchemaVersions() []string
}

type ParseResponse struct {
	Valid bool `json:"valid"`

	// When this exists, the payload has been sanitized and is considered safe to save
	Body []byte `json:"-"`

	// The payload parsed into a golang type
	Entity interface{} `json:"-"`

	// This includes potential errors and warnings
	Info []data.Notice `json:"info,omitempty"`

	// Some kinds may have more detailed response objects.  This can include line numbers etc
	Details interface{} `json:"details,omitempty"`
}

type IndexResponse struct {
	// Key value lookup for values that should be saved in the index.
	// NOTE: the keys must match field names defined in the kind `IndexStructure` response
	Fields map[string]interface{} `json:"fields,omitempty"`

	// Return a list of linked references
	References []EntityReference `json:"references,omitempty"`

	// ??? if we want text for search engine that is not actually saved
	// NonStoredSearchText string `json:"searchText,omitempty"`
}

type EntityReference struct {
	// UID may contain / to indicate parent folder's
	UID string `json:"UID,omitempty"`

	// dashboard, datasource, folder, etc
	Kind string `json:"kind,omitempty"`

	// prometheus etc
	Type string `json:"type,omitempty"`
}

// The Kind indicator for folders
const FolderKindID = "folder"

// FolderKind is the kind used to indicate a folder.  Folders do not have a body, but can include a listing
var FolderKind = NewGenericKind(KindInfo{
	ID:          FolderKindID,
	Description: "Folder",
})

type SecureValues map[string]string
