package resource

import (
	"embed"

	common "k8s.io/kube-openapi/pkg/common"
	openapi "k8s.io/kube-openapi/pkg/common"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

//go:embed query.schema.json query.definition.schema.json
var f embed.FS

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/grafana/grafana-plugin-sdk-go/backend.QueryDataResponse":                     schema_backend_query_data_response(ref),
		"github.com/grafana/grafana-plugin-sdk-go/data.Frame":                                    schema_data_frame(ref),
		"github.com/grafana/grafana-plugin-sdk-go/experimental/resource.GenericDataQuery":        schema_GenericQuery(ref),
		"github.com/grafana/grafana-plugin-sdk-go/experimental/resource.QueryTypeDefinitionSpec": schema_QueryTypeDefinitionSpec(ref),
	}
}

func schema_backend_query_data_response(_ common.ReferenceCallback) common.OpenAPIDefinition {
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

func schema_data_frame(_ common.ReferenceCallback) common.OpenAPIDefinition {
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

func schema_QueryTypeDefinitionSpec(_ common.ReferenceCallback) common.OpenAPIDefinition {
	s, _ := loadSchema("query.definition.schema.json")
	if s == nil {
		s = &spec.Schema{}
	}
	return common.OpenAPIDefinition{
		Schema: *s,
	}
}

func schema_GenericQuery(_ common.ReferenceCallback) common.OpenAPIDefinition {
	s, _ := GenericQuerySchema()
	if s == nil {
		s = &spec.Schema{}
	}
	s.SchemaProps.Type = []string{"object"}
	s.SchemaProps.AdditionalProperties = &spec.SchemaOrBool{Allows: true}
	return openapi.OpenAPIDefinition{Schema: *s}
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
