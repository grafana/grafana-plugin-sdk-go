package model

import (
	"encoding/json"
	"strings"
	"testing"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func TestDataSourceOpenAPIExtensionMarshalsQueries(t *testing.T) {
	value := DataSourceOpenAPIExtension{
		SecureValues: []SecureValueInfo{},
		Queries: &v0alpha1.QueryTypeDefinitionList{
			TypeMeta: v0alpha1.TypeMeta{
				Kind:       "QueryTypeDefinitionList",
				APIVersion: "datasource.grafana.app/v0alpha1",
			},
			Items: []v0alpha1.QueryTypeDefinition{{
				ObjectMeta: v0alpha1.ObjectMeta{
					Name: "Pull_Requests",
				},
				Spec: v0alpha1.QueryTypeDefinitionSpec{
					Discriminators: []v0alpha1.DiscriminatorFieldValue{{
						Field: "queryType",
						Value: "Pull_Requests",
					}},
					Schema: v0alpha1.JSONSchema{
						Spec: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Type: []string{"object"},
							},
						},
					},
					Examples: []v0alpha1.QueryExample{{
						Name: "Simple",
						SaveModel: v0alpha1.AsUnstructured(map[string]any{
							"queryType": "Pull_Requests",
						}),
					}},
				},
			}},
		},
	}

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	text := string(body)
	if !strings.Contains(text, `"queries"`) {
		t.Fatalf("expected queries field in output, got %s", text)
	}
	if !strings.Contains(text, `"kind":"QueryTypeDefinitionList"`) {
		t.Fatalf("expected query definition list kind in output, got %s", text)
	}
	if !strings.Contains(text, `"field":"queryType"`) {
		t.Fatalf("expected discriminator field in output, got %s", text)
	}
}
