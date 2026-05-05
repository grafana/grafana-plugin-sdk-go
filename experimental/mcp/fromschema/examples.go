package fromschema

import (
	"context"
	"encoding/json"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

// RegisterQueryExamples publishes a single resource "examples://query" listing
// every QueryExample, and attaches each example to the corresponding tool's
// Examples field. Must be called after RegisterQueryTools so the tools exist.
func RegisterQueryExamples(s *mcp.Server, schema *pluginschema.PluginSchema) {
	if schema == nil || schema.QueryExamples == nil {
		return
	}
	body, err := json.MarshalIndent(schema.QueryExamples, "", "  ")
	if err != nil {
		body = []byte("{}")
	}
	s.RegisterResource(mcp.Resource{
		URI:         "examples://query",
		Name:        "Query examples",
		Description: "Datasource-specific query examples grouped by queryType",
		MIMEType:    "application/json",
		Reader: func(_ context.Context) ([]byte, string, error) {
			return body, "application/json", nil
		},
	})

	// attach each example to its matching tool, if registered
	byName := map[string]*sdkapi.QueryExample{}
	for i := range schema.QueryExamples.Examples {
		ex := &schema.QueryExamples.Examples[i]
		byName[ex.QueryType] = ex
	}
	for _, t := range s.Tools() {
		qt, ok := t.Annotations["queryType"].(string)
		if !ok {
			continue
		}
		ex, ok := byName[qt]
		if !ok {
			continue
		}
		t.Examples = append(t.Examples, ex.SaveModel.Object)
		s.UpdateTool(t)
	}
}
