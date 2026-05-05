// Package fromschema turns a pluginschema.PluginSchema and a bound mcp.Server
// into registered tools/resources. Each Register* helper is independent and
// idempotent.
package fromschema

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
)

// RegisterHealthCheckTool adds a "check_health" tool that delegates to the
// CheckHealthHandler bound on the server. Safe to call before BindCheckHealthHandler;
// the tool will surface an error at call time if the handler is missing.
func RegisterHealthCheckTool(s *mcp.Server) {
	s.RegisterTool(mcp.Tool{
		Name:        "check_health",
		Description: "Run the datasource's health check",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return s.ExecuteHealthTool(ctx)
		},
	})
}
