package fromschema

import (
	"context"
	"encoding/json"
	"fmt"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

// RegisterQueryTools adds one MCP tool per QueryTypeDefinition in the schema.
// Each tool's name is "query_<discriminator-value>", its InputSchema is the
// query type's JSON Schema, and its handler delegates to the bound QueryDataHandler.
func RegisterQueryTools(s *mcp.Server, schema *pluginschema.PluginSchema) {
	if schema == nil || schema.QueryTypes == nil {
		return
	}
	for _, qt := range schema.QueryTypes.Items {
		discValue := qt.ObjectMeta.Name
		if len(qt.Spec.Discriminators) > 0 && qt.Spec.Discriminators[0].Value != "" {
			discValue = qt.Spec.Discriminators[0].Value
		}

		var inputSchema map[string]any
		if raw := schemaSpec(qt.Spec.Schema); len(raw) > 0 {
			_ = json.Unmarshal(raw, &inputSchema)
		}

		queryType := discValue
		s.RegisterTool(mcp.Tool{
			Name:        "query_" + discValue,
			Description: fmt.Sprintf("Run a %s query against the datasource", discValue),
			InputSchema: inputSchema,
			Annotations: map[string]any{"queryType": queryType},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return s.ExecuteQueryTool(ctx, queryType, args)
			},
		})
	}
}

// schemaSpec returns the JSON-serialized form of the JSON Schema. JSONSchema's
// MarshalJSON returns the inner schema body directly so this is just a wrapper.
func schemaSpec(s sdkapi.JSONSchema) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return b
}
