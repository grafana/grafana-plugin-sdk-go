package pluginspec_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginspec"
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
	sampleYAML, err := sample.ToYAML()
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

	compare := func(snapshotBytes []byte, format string) {
		snapshot, err := pluginspec.LoadSpec(snapshotBytes)
		if err != nil {
			writeFile = true
		} else {
			if diff := sample.Diff(snapshot); diff != "" {
				t.Errorf("%s results changed (-want +got):\n%s", format, diff)
				writeFile = true
			}
		}
	}

	if len(snapshotJSON) > 0 {
		compare(snapshotJSON, "JSON")
	}

	if len(snapshotYAML) > 0 {
		compare(snapshotYAML, "YAML")
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

func NewSample() (*pluginspec.OpenAPIExtension, error) {
	oas := &pluginspec.OpenAPIExtension{
		Settings: pluginspec.Settings{
			Spec: &spec.Schema{},

			SecureValues: []pluginspec.SecureValueInfo{
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
	p := oas.Settings.Spec
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

	oas.Routes = &pluginspec.Routes{
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
