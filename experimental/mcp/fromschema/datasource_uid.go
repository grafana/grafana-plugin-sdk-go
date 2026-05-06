package fromschema

const datasourceUIDProp = "datasource_uid"

// addDatasourceUID injects a "datasource_uid" property into an MCP tool's
// InputSchema. This allows MCP clients to specify which Grafana datasource
// instance to target. The parameter is optional: when omitted and only one
// instance is registered, the server uses it automatically.
func addDatasourceUID(schema map[string]any) map[string]any {
	if schema == nil {
		schema = map[string]any{"type": "object"}
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
		schema["properties"] = props
	}
	props[datasourceUIDProp] = map[string]any{
		"type":        "string",
		"description": "UID of the Grafana datasource instance to query. Omit when only one instance is configured.",
	}
	return schema
}
