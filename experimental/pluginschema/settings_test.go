package pluginschema_test

// func TestReadSampleConfig(t *testing.T) {
// 	sample, err := NewSample()
// 	if err != nil {
// 		t.Fatalf("TestdataOpenAPIExtension() error = %v", err)
// 	}

// 	sampleJSON, err := json.MarshalIndent(sample, "", "  ")
// 	if err != nil {
// 		t.Fatalf("failed to marshal result: %v", err)
// 	}
// 	sampleYAML, err := pluginschema.ToYAML(sample)
// 	if err != nil {
// 		t.Fatalf("failed to marshal result: %v", err)
// 	}

// 	snapshotPathJSON := filepath.Join("testdata", "v0alpha1", "settings.json")
// 	snapshotPathYAML := filepath.Join("testdata", "v0alpha1", "settings.yaml")
// 	snapshotDir := filepath.Dir(snapshotPathJSON)

// 	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
// 		t.Fatalf("failed to create testdata directory: %v", err)
// 	}

// 	writeFile := false
// 	snapshotJSON, err := os.ReadFile(snapshotPathJSON) // #nosec G304
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			writeFile = true
// 			t.Error("unable to read snapshot")
// 		} else {
// 			t.Fatalf("failed to read snapshot: %v", err)
// 		}
// 	}

// 	snapshotYAML, err := os.ReadFile(snapshotPathYAML) // #nosec G304
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			writeFile = true
// 			t.Error("unable to read snapshot")
// 		} else {
// 			t.Fatalf("failed to read snapshot: %v", err)
// 		}
// 	}

// 	compare := func(snapshotBytes []byte, format string) {
// 		snapshot := &pluginschema.Settings{}
// 		err := pluginschema.Load(snapshotBytes, snapshot)
// 		if err != nil {
// 			writeFile = true
// 		} else {
// 			if diff := pluginschema.Diff(sample, snapshot); diff != "" {
// 				t.Errorf("%s results changed (-want +got):\n%s", format, diff)
// 				writeFile = true
// 			}
// 		}
// 	}

// 	if len(snapshotJSON) > 0 {
// 		compare(snapshotJSON, "JSON")
// 	}

// 	if len(snapshotYAML) > 0 {
// 		compare(snapshotYAML, "YAML")
// 	}

// 	if writeFile {
// 		if err := os.WriteFile(snapshotPathJSON, sampleJSON, 0600); err != nil {
// 			t.Fatalf("failed to update snapshot: %v", err)
// 		}
// 		if err := os.WriteFile(snapshotPathYAML, sampleYAML, 0600); err != nil {
// 			t.Fatalf("failed to update snapshot: %v", err)
// 		}
// 		t.Fatal("snapshot mismatch, snapshot updated")
// 	}
// }

// func NewSample() (*pluginschema.Settings, error) {

// 	// // Resource routes
// 	// // https://github.com/grafana/grafana/blob/main/pkg/tsdb/grafana-testdata-datasource/resource_handler.go#L20
// 	// unstructured := spec.MapProperty(nil)
// 	// unstructuredResponse := &spec3.Responses{
// 	// 	ResponsesProps: spec3.ResponsesProps{
// 	// 		Default: &spec3.Response{
// 	// 			ResponseProps: spec3.ResponseProps{
// 	// 				Content: map[string]*spec3.MediaType{
// 	// 					"application/json": {
// 	// 						MediaTypeProps: spec3.MediaTypeProps{
// 	// 							Schema: unstructured,
// 	// 						},
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 	},
// 	// }
// 	// unstructuredRequest := &spec3.RequestBody{
// 	// 	RequestBodyProps: spec3.RequestBodyProps{
// 	// 		Content: map[string]*spec3.MediaType{
// 	// 			"application/json": {
// 	// 				MediaTypeProps: spec3.MediaTypeProps{
// 	// 					Schema: unstructured,
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 	},
// 	// }

// 	// oas.Routes = &pluginschema.Routes{
// 	// 	Resource: map[string]*spec3.Path{
// 	// 		"": {
// 	// 			PathProps: spec3.PathProps{
// 	// 				Summary: "hello world",
// 	// 				Get: &spec3.Operation{
// 	// 					OperationProps: spec3.OperationProps{
// 	// 						Responses: &spec3.Responses{
// 	// 							ResponsesProps: spec3.ResponsesProps{
// 	// 								Default: &spec3.Response{
// 	// 									ResponseProps: spec3.ResponseProps{
// 	// 										Content: map[string]*spec3.MediaType{
// 	// 											"text/plain": {
// 	// 												MediaTypeProps: spec3.MediaTypeProps{
// 	// 													Schema: spec.StringProperty(),
// 	// 												},
// 	// 											},
// 	// 										},
// 	// 									},
// 	// 								},
// 	// 							},
// 	// 						},
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 		"/stream": {
// 	// 			PathProps: spec3.PathProps{
// 	// 				Summary: "Get streaming response",
// 	// 				Get: &spec3.Operation{
// 	// 					OperationProps: spec3.OperationProps{
// 	// 						Parameters: []*spec3.Parameter{
// 	// 							{
// 	// 								ParameterProps: spec3.ParameterProps{
// 	// 									Name:        "count",
// 	// 									In:          "query",
// 	// 									Schema:      spec.Int64Property(),
// 	// 									Description: "number of points that will be returned",
// 	// 									Example:     10,
// 	// 								},
// 	// 							},
// 	// 							{
// 	// 								ParameterProps: spec3.ParameterProps{
// 	// 									Name:        "start",
// 	// 									In:          "query",
// 	// 									Schema:      spec.Int64Property(),
// 	// 									Description: "the start value",
// 	// 								},
// 	// 							},
// 	// 						},
// 	// 						Responses: unstructuredResponse,
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 		"/test": {
// 	// 			PathProps: spec3.PathProps{
// 	// 				Summary: "Echo any request",
// 	// 				Post: &spec3.Operation{
// 	// 					OperationProps: spec3.OperationProps{
// 	// 						RequestBody: unstructuredRequest,
// 	// 						Responses:   unstructuredResponse,
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 	},
// 	// 	Proxy: map[string]*spec3.Path{
// 	// 		"": {
// 	// 			PathProps: spec3.PathProps{
// 	// 				Summary: "simple proxy",
// 	// 				Get: &spec3.Operation{
// 	// 					OperationProps: spec3.OperationProps{
// 	// 						Responses: &spec3.Responses{
// 	// 							ResponsesProps: spec3.ResponsesProps{
// 	// 								Default: &spec3.Response{
// 	// 									ResponseProps: spec3.ResponseProps{
// 	// 										Content: map[string]*spec3.MediaType{
// 	// 											"text/plain": {
// 	// 												MediaTypeProps: spec3.MediaTypeProps{
// 	// 													Schema: spec.StringProperty(),
// 	// 												},
// 	// 											},
// 	// 										},
// 	// 									},
// 	// 								},
// 	// 							},
// 	// 						},
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 		"/with-path": {
// 	// 			PathProps: spec3.PathProps{
// 	// 				Summary: "proxy with path",
// 	// 				Get: &spec3.Operation{
// 	// 					OperationProps: spec3.OperationProps{
// 	// 						Parameters: []*spec3.Parameter{
// 	// 							{
// 	// 								ParameterProps: spec3.ParameterProps{
// 	// 									Name:        "count",
// 	// 									In:          "query",
// 	// 									Schema:      spec.Int64Property(),
// 	// 									Description: "number of points that will be returned",
// 	// 									Example:     10,
// 	// 								},
// 	// 							},
// 	// 						},
// 	// 						Responses: unstructuredResponse,
// 	// 					},
// 	// 				},
// 	// 			},
// 	// 		},
// 	// 	},
// 	//}
// 	return &settings, nil
// }
