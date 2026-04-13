package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/yaml"
)

func TestReadSampleConfig(t *testing.T) {
	sample, err := NewSample()
	if err != nil {
		t.Fatalf("TestdataOpenAPIExtension() error = %v", err)
	}

	sampleJSON, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	sampleYAML, err := yaml.Marshal(sample)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	snapshotPathJSON := filepath.Join("testdata", "sample.json")
	snapshotPathYAML := filepath.Join("testdata", "sample.yaml")
	snapshotDir := filepath.Dir(snapshotPathJSON)

	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		t.Fatalf("failed to create testdata directory: %v", err)
	}

	writeFile := false
	snapshotJSON, err := os.ReadFile(snapshotPathJSON) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			writeFile = true
			t.Error("unable to read snapshot")
		} else {
			t.Fatalf("failed to read snapshot: %v", err)
		}
	}

	snapshotYAML, err := os.ReadFile(snapshotPathYAML) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			writeFile = true
			t.Error("unable to read snapshot")
		} else {
			t.Fatalf("failed to read snapshot: %v", err)
		}
	}

	if len(snapshotJSON) > 0 {
		if !assert.JSONEq(t, string(sampleJSON), string(snapshotJSON)) {
			writeFile = true
			t.Error("snapshot changed")
		}
	}

	if len(snapshotYAML) > 0 {
		snapshot := &OpenAPIExtension{}
		err := yaml.Unmarshal(snapshotYAML, snapshot)
		if err != nil {
			writeFile = true
		} else {
			if diff := cmp.Diff(sample, snapshot,
				AlwaysCompareNumeric,
				cmpopts.EquateApprox(0.001, 0.0001),
				cmp.Comparer(func(a, b spec.Ref) bool {
					return a.String() == b.String()
				})); diff != "" {
				t.Errorf("Yaml results changed (-want +got):\n%s", diff)
				writeFile = true
			}
		}
	}

	if writeFile {
		if err := os.WriteFile(snapshotPathJSON, sampleJSON, 0600); err != nil {
			t.Fatalf("failed to update snapshot: %v", err)
		}
		if err := os.WriteFile(snapshotPathYAML, sampleYAML, 0600); err != nil {
			t.Fatalf("failed to update snapshot: %v", err)
		}
		t.Fatal("snapshot mismatch, snapshot updated")
	}
}

func NewSample() (*OpenAPIExtension, error) {
	oas := &OpenAPIExtension{
		Settings: Settings{
			Schema: &spec.Schema{},

			SecureValues: []SecureValueInfo{
				{
					Key:         "aaa",
					Description: "describe aaa",
					Required:    true,
				}, {
					Key:         "bbb",
					Description: "describe bbb",
				},
			},

			Examples: map[string]*spec3.Example{
				"": { // empty is the default one displayed in swagger
					ExampleProps: spec3.ExampleProps{
						Summary: "Empty testdata",
						Value: map[string]any{
							"kind": "DataSource",
							"metadata": map[string]any{
								"name": "my-testdata-datasource",
							},
							"spec": map[string]any{
								"title": "My TestData Datasource",
							},
						},
					},
				},
				"with-url": {
					ExampleProps: spec3.ExampleProps{
						Summary: "Testdata with URL (not used)",
						Value: map[string]any{
							"kind": "DataSource",
							"metadata": map[string]any{
								"name": "testdata-with-url",
							},
							"spec": map[string]any{
								"title": "TestData with URL",
								"url":   "http://example.com",
							},
						},
					},
				},
			},
		},
	}

	// Dummy spec
	p := oas.Settings.Schema
	p.Description = "Test data does not require any explicit configuration"
	p.Required = []string{"title"}
	p.AdditionalProperties = &spec.SchemaOrBool{Allows: false}
	p.Properties = map[string]spec.Schema{
		"title": *spec.StringProperty().WithDescription("display name"),
		"url":   *spec.StringProperty().WithDescription("not used"),
	}
	p.Example = map[string]any{
		"url": "http://xxxx",
	}

	// Resource routes
	// https://github.com/grafana/grafana/blob/main/pkg/tsdb/grafana-testdata-datasource/resource_handler.go#L20
	unstructured := spec.MapProperty(nil)
	unstructuredResponse := &spec3.Responses{
		ResponsesProps: spec3.ResponsesProps{
			Default: &spec3.Response{
				ResponseProps: spec3.ResponseProps{
					Content: map[string]*spec3.MediaType{
						"application/json": {
							MediaTypeProps: spec3.MediaTypeProps{
								Schema: unstructured,
							},
						},
					},
				},
			},
		},
	}
	unstructuredRequest := &spec3.RequestBody{
		RequestBodyProps: spec3.RequestBodyProps{
			Content: map[string]*spec3.MediaType{
				"application/json": {
					MediaTypeProps: spec3.MediaTypeProps{
						Schema: unstructured,
					},
				},
			},
		},
	}

	oas.Routes = &Routes{
		Resource: map[string]*spec3.Path{
			"": {
				PathProps: spec3.PathProps{
					Summary: "hello world",
					Get: &spec3.Operation{
						OperationProps: spec3.OperationProps{
							Responses: &spec3.Responses{
								ResponsesProps: spec3.ResponsesProps{
									Default: &spec3.Response{
										ResponseProps: spec3.ResponseProps{
											Content: map[string]*spec3.MediaType{
												"text/plain": {
													MediaTypeProps: spec3.MediaTypeProps{
														Schema: spec.StringProperty(),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/stream": {
				PathProps: spec3.PathProps{
					Summary: "Get streaming response",
					Get: &spec3.Operation{
						OperationProps: spec3.OperationProps{
							Parameters: []*spec3.Parameter{
								{
									ParameterProps: spec3.ParameterProps{
										Name:        "count",
										In:          "query",
										Schema:      spec.Int64Property(),
										Description: "number of points that will be returned",
										Example:     10,
									},
								},
								{
									ParameterProps: spec3.ParameterProps{
										Name:        "start",
										In:          "query",
										Schema:      spec.Int64Property(),
										Description: "the start value",
									},
								},
							},
							Responses: unstructuredResponse,
						},
					},
				},
			},
			"/test": {
				PathProps: spec3.PathProps{
					Summary: "Echo any request",
					Post: &spec3.Operation{
						OperationProps: spec3.OperationProps{
							RequestBody: unstructuredRequest,
							Responses:   unstructuredResponse,
						},
					},
				},
			},
		},
		Proxy: map[string]*spec3.Path{
			"": {
				PathProps: spec3.PathProps{
					Summary: "simple proxy",
					Get: &spec3.Operation{
						OperationProps: spec3.OperationProps{
							Responses: &spec3.Responses{
								ResponsesProps: spec3.ResponsesProps{
									Default: &spec3.Response{
										ResponseProps: spec3.ResponseProps{
											Content: map[string]*spec3.MediaType{
												"text/plain": {
													MediaTypeProps: spec3.MediaTypeProps{
														Schema: spec.StringProperty(),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/with-path": {
				PathProps: spec3.PathProps{
					Summary: "proxy with path",
					Get: &spec3.Operation{
						OperationProps: spec3.OperationProps{
							Parameters: []*spec3.Parameter{
								{
									ParameterProps: spec3.ParameterProps{
										Name:        "count",
										In:          "query",
										Schema:      spec.Int64Property(),
										Description: "number of points that will be returned",
										Example:     10,
									},
								},
							},
							Responses: unstructuredResponse,
						},
					},
				},
			},
		},
	}
	return oas, nil
}

// AlwaysCompareNumeric transforms all ints and floats to float64 for comparison
var AlwaysCompareNumeric = cmp.FilterValues(func(x, y any) bool {
	return isNumeric(x) && isNumeric(y)
}, cmp.Transformer("NormalizeNumeric", func(v any) float64 {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case float64:
		return t
	default:
		return 0 // Should be filtered by isNumeric
	}
}))

func isNumeric(v any) bool {
	switch v.(type) {
	case int, int64, float64:
		return true
	default:
		return false
	}
}
