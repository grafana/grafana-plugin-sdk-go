package entity

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type PlainTextEntity struct {
	Envelope

	Body string `json:"body,omitempty"`
}

var _ Kind = &PlainTextKind{}

type PlainTextKind struct {
	info KindInfo
}

func NewPlainTextKind(info KindInfo) *PlainTextKind {
	return &PlainTextKind{info: info}
}

func (k *PlainTextKind) Info() KindInfo {
	k.info.IsRaw = true
	return k.info
}

func (k *PlainTextKind) GoType() interface{} {
	return &PlainTextEntity{}
}

func (k *PlainTextKind) Read(payload []byte) (interface{}, error) {
	// ?? make sure the payload is safe string bytes?
	g := &PlainTextEntity{}
	g.Body = string(payload)
	return g, nil
}

func (k *PlainTextKind) Write(v interface{}) ([]byte, error) {
	g, ok := v.(*PlainTextEntity)
	if !ok {
		return nil, fmt.Errorf("expected RawFileEntity")
	}
	return []byte(g.Body), nil
}

func (k *PlainTextKind) Validate(payload []byte, details bool) ValidationResponse {
	_, err := k.Read(payload)
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
		Result: payload,
	}
}

func (k *PlainTextKind) Migrate(payload []byte, targetVersion string) ValidationResponse {
	return k.Validate(payload, false) // migration is a noop
}

func (k *PlainTextKind) GetSchemaVersions() []string {
	return nil
}

func (k *PlainTextKind) GetJSONSchema(schemaVersion string) []byte {
	// The payload is not json!
	return nil
}
