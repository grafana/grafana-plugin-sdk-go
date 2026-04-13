package querygen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

func TestBuildDefinitionsMatchLocalGoldenForMultipleQueries(t *testing.T) {
	dir := writeCompatFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

// QueryKind identifies which query variant to run.
type QueryKind string

const (
	// Run math expressions.
	QueryKindMath QueryKind = "math"
	// Run reduce expressions.
	QueryKindReduce QueryKind = "reduce"
)

type MathQuery struct {
	QueryType QueryKind ` + "`json:\"queryType\"`" + `
	Expression string ` + "`json:\"expression\"`" + `
}

type ReduceQuery struct {
	QueryType QueryKind ` + "`json:\"queryType\"`" + `
	Input string ` + "`json:\"input\"`" + `
}
`,
	})

	definitions, err := BuildDefinitionsInModule(RuntimeOptions{
		Dir:      dir,
		PluginID: []string{"fixture-datasource"},
	}, []RuntimeRegistration{
		{
			PackagePath: "fixture/pkg/models",
			TypeName:    "MathQuery",
			Name:        "math",
			Discriminators: []v0alpha1.DiscriminatorFieldValue{{
				Field: "queryType",
				Value: "math",
			}},
			Examples: []v0alpha1.QueryExample{{
				Name: "basic math",
				SaveModel: v0alpha1.AsUnstructured(map[string]any{
					"queryType":  "math",
					"expression": "$A + 1",
				}),
			}},
		},
		{
			PackagePath: "fixture/pkg/models",
			TypeName:    "ReduceQuery",
			Name:        "reduce",
			Discriminators: []v0alpha1.DiscriminatorFieldValue{{
				Field: "queryType",
				Value: "reduce",
			}},
			Changelog: []string{"Added reducer input support."},
		},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	assertGoldenJSON(t, definitions, `
{
  "kind": "QueryTypeDefinitionList",
  "apiVersion": "datasource.grafana.app/v0alpha1",
  "metadata": {},
  "items": [
    {
      "metadata": {
        "name": "math"
      },
      "spec": {
        "discriminators": [
          {
            "field": "queryType",
            "value": "math"
          }
        ],
        "schema": {
          "$schema": "https://json-schema.org/draft-04/schema",
          "additionalProperties": false,
          "properties": {
            "expression": {
              "type": "string"
            },
            "queryType": {
              "description": "QueryKind identifies which query variant to run.\n\nPossible enum values:\n - \u0060\"math\"\u0060 Run math expressions.\n - \u0060\"reduce\"\u0060 Run reduce expressions.",
              "enum": [
                "math",
                "reduce"
              ],
              "type": "string"
            }
          },
          "type": "object"
        },
        "examples": [
          {
            "name": "basic math",
            "saveModel": {
              "expression": "$A + 1",
              "queryType": "math"
            }
          }
        ]
      }
    },
    {
      "metadata": {
        "name": "reduce"
      },
      "spec": {
        "discriminators": [
          {
            "field": "queryType",
            "value": "reduce"
          }
        ],
        "schema": {
          "$schema": "https://json-schema.org/draft-04/schema",
          "additionalProperties": false,
          "properties": {
            "input": {
              "type": "string"
            },
            "queryType": {
              "description": "QueryKind identifies which query variant to run.\n\nPossible enum values:\n - \u0060\"math\"\u0060 Run math expressions.\n - \u0060\"reduce\"\u0060 Run reduce expressions.",
              "enum": [
                "math",
                "reduce"
              ],
              "type": "string"
            }
          },
          "type": "object"
        },
        "examples": null,
        "changelog": [
          "Added reducer input support."
        ]
      }
    }
  ]
}
`)
}

func TestBuildDefinitionsMatchLocalGoldenForSingleQuery(t *testing.T) {
	dir := writeCompatFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

// Query is the saved query model.
type Query struct {
	QueryType string ` + "`json:\"queryType,omitempty\"`" + `
	Expression string ` + "`json:\"expression\"`" + `
}
`,
	})

	definitions, err := BuildDefinitionsInModule(RuntimeOptions{
		Dir:      dir,
		PluginID: []string{"fixture-datasource"},
	}, []RuntimeRegistration{{
		PackagePath: "fixture/pkg/models",
		TypeName:    "Query",
		Name:        "math",
		Discriminators: []v0alpha1.DiscriminatorFieldValue{{
			Field: "queryType",
			Value: "math",
		}},
	}})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	assertGoldenJSON(t, definitions, `
{
  "kind": "QueryTypeDefinitionList",
  "apiVersion": "datasource.grafana.app/v0alpha1",
  "metadata": {},
  "items": [
    {
      "metadata": {
        "name": "math"
      },
      "spec": {
        "discriminators": [
          {
            "field": "queryType",
            "value": "math"
          }
        ],
        "schema": {
          "$schema": "https://json-schema.org/draft-04/schema",
          "additionalProperties": false,
          "description": "Query is the saved query model.",
          "properties": {
            "expression": {
              "type": "string"
            },
            "queryType": {
              "type": "string"
            }
          },
          "type": "object"
        },
        "examples": null
      }
    }
  ]
}
`)
}

func assertGoldenJSON(t *testing.T, actual any, expected string) {
	t.Helper()

	var actualValue any
	body, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal actual failed: %v", err)
	}
	if err := json.Unmarshal(body, &actualValue); err != nil {
		t.Fatalf("unmarshal actual failed: %v", err)
	}

	var expectedValue any
	if err := json.Unmarshal([]byte(strings.TrimSpace(expected)), &expectedValue); err != nil {
		t.Fatalf("unmarshal expected failed: %v", err)
	}

	if !reflect.DeepEqual(actualValue, expectedValue) {
		actualPretty, _ := json.MarshalIndent(actualValue, "", "  ")
		expectedPretty, _ := json.MarshalIndent(expectedValue, "", "  ")
		t.Fatalf("golden mismatch\nexpected:\n%s\nactual:\n%s", expectedPretty, actualPretty)
	}
}

func writeCompatFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir failed for %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(strings.TrimLeft(content, "\n")), 0o644); err != nil {
			t.Fatalf("write failed for %s: %v", fullPath, err)
		}
	}

	return dir
}
