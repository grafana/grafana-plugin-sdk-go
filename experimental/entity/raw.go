package entity

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type RawFileEntity struct {
	Envelope

	Body []byte `json:"body,omitempty"`
}

var _ Kind = &RawFileKind{}

type RawFileSanitizer = func(payload []byte) ([]byte, error)

type RawFileKind struct {
	info     KindInfo
	sanitize RawFileSanitizer
}

func NewRawFileKind(info KindInfo, sanitize RawFileSanitizer) *RawFileKind {
	return &RawFileKind{
		info:     info,
		sanitize: sanitize,
	}
}

func (k *RawFileKind) Info() KindInfo {
	k.info.IsRaw = true
	return k.info
}

func (k *RawFileKind) GoType() interface{} {
	return &RawFileEntity{}
}

func (k *RawFileKind) Read(payload []byte) (interface{}, error) {
	g := &RawFileEntity{}
	g.Kind = k.info.ID
	g.Body = payload
	return g, nil
}

func (k *RawFileKind) Write(v interface{}) ([]byte, error) {
	g, ok := v.(*RawFileEntity)
	if !ok {
		return nil, fmt.Errorf("expected RawFileEntity")
	}
	return g.Body, nil
}

func (k *RawFileKind) GetReferences(v interface{}) []EntityLocator {
	return nil
}

func (k *RawFileKind) Validate(payload []byte, details bool) ValidationResponse {
	out, err := k.sanitize(payload)
	if err != nil {
		return ValidationResponse{
			Valid: false,
			Info: []data.Notice{
				{
					Severity: data.NoticeSeverityError,
					Text:     err.Error(),
				},
			},
		}
	}
	return ValidationResponse{
		Valid:  true,
		Result: out,
	}
}

func (k *RawFileKind) Migrate(payload []byte, targetVersion string) ValidationResponse {
	return k.Validate(payload, false) // migration is a noop
}

func (k *RawFileKind) GetSchemaVersions() []string {
	return nil
}

func (k *RawFileKind) GetJSONSchema(schemaVersion string) []byte {
	// The payload is not json!
	return nil
}
