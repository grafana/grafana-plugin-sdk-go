package standard

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/entity"
)

type GenericEntity struct {
	entity.Envelope

	Body interface{} `json:"body,omitempty"`
}

var _ entity.Kind = &GenericKind{}

type GenericKind struct {
	info entity.KindInfo
}

func NewGenericKind(info entity.KindInfo) *GenericKind {
	return &GenericKind{info: info}
}

func (k *GenericKind) Info() entity.KindInfo {
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

func (k *GenericKind) Validate(payload []byte, details bool) entity.ValidationResponse {
	g, err := k.Read(payload)
	if err == nil {
		// pretty print the payload
		payload, err = json.MarshalIndent(g, "", "  ")
	}

	if err != nil {
		return entity.ValidationResponse{
			Valid: false,
			Info: []data.Notice{
				{
					Severity: data.NoticeSeverityError,
					Text:     err.Error(),
				},
			},
		}
	}

	return entity.ValidationResponse{
		Valid:  true,
		Result: payload,
	}
}

func (k *GenericKind) Migrate(payload []byte, targetVersion string) entity.ValidationResponse {
	return k.Validate(payload, false) // migration is a noop
}

func (k *GenericKind) GetSchemaVersions() []string {
	return nil
}

func (k *GenericKind) GetJSONSchema(schemaVersion string) []byte {
	return []byte(`{
		"$id": "https://example.com/person.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Generic Entity envelope",
		"type": "object",
		"properties": {
		  "uid": {
			"type": "string",
			"description": "The person's first name."
		  },
		  "kind": {
			"type": "string",
			"description": "The person's last name."
		  },
		  "schemaVersion": {
			"type": "string",
			"description": "Age in years which must be equal to or greater than zero."
		  }
		  "body": {
			"type": "object",
			"description": "Any object"
		  }
		}
	  }`)
}
