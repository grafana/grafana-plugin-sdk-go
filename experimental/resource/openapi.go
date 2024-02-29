package resource

import (
	"embed"

	"k8s.io/kube-openapi/pkg/common"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

//go:embed query.schema.json query.definition.schema.json
var f embed.FS

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/grafana/grafana-plugin-sdk-go/backend.QueryDataResponse":                     schemaQueryDataResponse(ref),
		"github.com/grafana/grafana-plugin-sdk-go/data.Frame":                                    schemaDataFrame(ref),
		"github.com/grafana/grafana-plugin-sdk-go/experimental/resource.GenericDataQuery":        schemaGenericQuery(ref),
		"github.com/grafana/grafana-plugin-sdk-go/experimental/resource.QueryTypeDefinitionSpec": schemaQueryTypeDefinitionSpec(ref),
	}
}

func schemaQueryDataResponse(_ common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "results keyed by refId",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"results": *spec.MapProperty(&spec.Schema{
						SchemaProps: spec.SchemaProps{
							Description:          "any object for now",
							Type:                 []string{"object"},
							Properties:           map[string]spec.Schema{},
							AdditionalProperties: &spec.SchemaOrBool{Allows: true},
						},
					}),
				},
				AdditionalProperties: &spec.SchemaOrBool{Allows: false},
			},
		},
	}
}

func schemaDataFrame(_ common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description:          "any object for now",
				Type:                 []string{"object"},
				Properties:           map[string]spec.Schema{},
				AdditionalProperties: &spec.SchemaOrBool{Allows: true},
			},
		},
	}
}

func schemaQueryTypeDefinitionSpec(_ common.ReferenceCallback) common.OpenAPIDefinition {
	s, _ := loadSchema("query.definition.schema.json")
	if s == nil {
		s = &spec.Schema{}
	}
	return common.OpenAPIDefinition{
		Schema: *s,
	}
}

func schemaGenericQuery(_ common.ReferenceCallback) common.OpenAPIDefinition {
	s, _ := GenericQuerySchema()
	if s == nil {
		s = &spec.Schema{}
	}
	s.SchemaProps.Type = []string{"object"}
	s.SchemaProps.AdditionalProperties = &spec.SchemaOrBool{Allows: true}
	return common.OpenAPIDefinition{Schema: *s}
}

// Get the cached feature list (exposed as a k8s resource)
func GenericQuerySchema() (*spec.Schema, error) {
	return loadSchema("query.schema.json")
}

// Get the cached feature list (exposed as a k8s resource)
func loadSchema(path string) (*spec.Schema, error) {
	body, err := f.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := &spec.Schema{}
	err = s.UnmarshalJSON(body)
	return s, err
}
