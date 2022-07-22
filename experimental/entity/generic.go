package entity

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type GenericEntity struct {
	Envelope

	Body interface{} `json:"body,omitempty"`
}

var _ Kind = &GenericKind{}

type GenericKind struct {
	info *KindInfo
}

func NewGenericKind(info *KindInfo) *GenericKind {
	return &GenericKind{info: info}
}

func (k *GenericKind) Info() *KindInfo {
	return k.info
}

func (k *GenericKind) GoType() interface{} {
	return &GenericEntity{}
}

func (k *GenericKind) Read(payload []byte) (interface{}, error) {
	g := &GenericEntity{}
	err := json.Unmarshal(payload, g)
	if err != nil {
		return nil, err
	}
	if g.Kind == "" {
		g.Kind = k.info.ID
	} else if g.Kind != k.info.ID {
		return nil, fmt.Errorf("expected kind: %s", k.info.ID)
	}
	return g, nil
}

func (k *GenericKind) Write(v interface{}) ([]byte, error) {
	g, ok := v.(*GenericEntity)
	if !ok {
		return nil, fmt.Errorf("expected RawFileEntity")
	}
	return json.MarshalIndent(g, "", "  ")
}

func (k *GenericKind) GetReferences(v interface{}) []EntityLocator {
	return nil
}

func (k *GenericKind) Normalize(payload []byte, details bool) NormalizeResponse {
	g, err := k.Read(payload)
	if err == nil {
		// pretty print the payload
		payload, err = json.MarshalIndent(g, "", "  ")
	}

	if err != nil {
		return NormalizeResponse{
			Valid: false,
			Info: []data.Notice{
				{
					Severity: data.NoticeSeverityError,
					Text:     err.Error(),
				},
			},
		}
	}

	return NormalizeResponse{
		Valid:  true,
		Result: payload,
	}
}

func (k *GenericKind) Migrate(payload []byte, targetVersion string) NormalizeResponse {
	return k.Normalize(payload, false) // migration is a noop
}

func (k *GenericKind) GetSchemaVersions() []string {
	return nil
}

func (k *GenericKind) GetJSONSchema(schemaVersion string) []byte {
	return GetEnvelopeJSON(k.info.ID)
}

func GetEnvelopeJSON(kind string) []byte {
	kindRule := `{
		"type": "string",
		"description": "Entity kind identifier"
	  }`
	if kind != "" {
		kindRule = `{
			"type": "string",
			"pattern": "^` + kind + `$"
		  }`
	}

	return []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Entity envelope",
		"type": "object",
		"properties": {
		  "uid": {
			"type": "string",
			"description": "System identifier."
		  },
		  "kind": ` + kindRule + `,
		  "schemaVersion": {
			"type": "string",
			"description": "The schema used to validate json messages (including the envelope)"
		  }
		  "body": {
			"type": "object",
			"description": "Any object"
		  }
		}
	  }`)
}