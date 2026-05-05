package fromschema

import (
	"context"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"k8s.io/kube-openapi/pkg/spec3"
)

// RegisterRouteTools adds one MCP tool per (path, method) pair under /resources/*.
// /proxy/* paths are skipped.
//
// The /resources/ prefix is stripped both from the tool name and from the path
// passed to CallResourceHandler at runtime - Grafana strips that prefix before
// forwarding resource calls to the plugin, so the plugin's handler sees e.g.
// "/labels", not "/resources/labels".
func RegisterRouteTools(s *mcp.Server, schema *pluginschema.PluginSchema) {
	if schema == nil || schema.Routes == nil {
		return
	}
	for schemaPath, p := range schema.Routes.Paths {
		if !strings.HasPrefix(schemaPath, "/resources/") {
			continue
		}
		runtimePath := strings.TrimPrefix(schemaPath, "/resources")
		registerOpIfPresent(s, "GET", runtimePath, p.Get)
		registerOpIfPresent(s, "POST", runtimePath, p.Post)
		registerOpIfPresent(s, "PUT", runtimePath, p.Put)
		registerOpIfPresent(s, "DELETE", runtimePath, p.Delete)
		registerOpIfPresent(s, "PATCH", runtimePath, p.Patch)
	}
}

func registerOpIfPresent(s *mcp.Server, method, path string, op *spec3.Operation) {
	if op == nil {
		return
	}
	spec := mcp.RouteToolSpec{
		Method: method,
		Path:   path,
	}
	inputProps := map[string]any{}
	required := []string{}
	for _, param := range op.Parameters {
		if param == nil {
			continue
		}
		switch param.In {
		case "path":
			spec.PathParams = append(spec.PathParams, param.Name)
		case "query":
			spec.QueryArgs = append(spec.QueryArgs, param.Name)
		}
		// add to input schema regardless of `In` so the tool exposes it
		inputProps[param.Name] = paramToSchema(param)
		if param.Required {
			required = append(required, param.Name)
		}
	}
	if op.RequestBody != nil {
		spec.BodyArg = "body"
		inputProps["body"] = map[string]any{"type": "object"}
		required = append(required, "body")
	}

	inputSchema := map[string]any{
		"type":       "object",
		"properties": inputProps,
	}
	if len(required) > 0 {
		inputSchema["required"] = required
	}

	desc := op.Summary
	if desc == "" {
		desc = op.Description
	}
	if desc == "" {
		desc = method + " " + path
	}

	s.RegisterTool(mcp.Tool{
		Name:        toolName(method, path),
		Description: desc,
		InputSchema: inputSchema,
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return s.ExecuteRouteTool(ctx, spec, args)
		},
	})
}

func paramToSchema(p *spec3.Parameter) map[string]any {
	out := map[string]any{}
	if p.Schema != nil && len(p.Schema.Type) > 0 {
		out["type"] = p.Schema.Type[0]
	} else {
		out["type"] = "string"
	}
	if p.Description != "" {
		out["description"] = p.Description
	}
	return out
}

// toolName converts (GET, "/labels") to "get_labels". Path params are dropped
// from the name (their values come from tool args).
func toolName(method, path string) string {
	m := strings.ToLower(method)
	parts := []string{m}
	for _, seg := range strings.Split(strings.Trim(path, "/"), "/") {
		if seg == "" || strings.HasPrefix(seg, "{") {
			continue
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, "_")
}
