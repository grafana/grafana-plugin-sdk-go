package mcp

import "context"

// ToolHandler runs the tool. args is a JSON object decoded from the MCP call.
// It returns the tool output (will be JSON-encoded as TextContent) or an error.
type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

// Tool is a registerable MCP tool.
type Tool struct {
	Name        string
	Description string
	// InputSchema is a JSON Schema document (as a map) describing the tool's input.
	// Pass nil for tools that take no arguments.
	InputSchema map[string]any
	// Annotations is a free-form metadata map exposed to MCP clients.
	Annotations map[string]any
	// Examples surface as the MCP tool's "examples" field.
	Examples []any
	// Handler is the function invoked on tools/call. Required.
	Handler ToolHandler
}

// RegisterTool adds a tool. If a tool with the same Name already exists, it
// is replaced (UpdateTool semantics).
func (s *Server) RegisterTool(t Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.tools {
		if existing.Name == t.Name {
			s.tools[i] = t
			return
		}
	}
	s.tools = append(s.tools, t)
}

// UpdateTool is an alias for RegisterTool to make the override intent explicit
// when callers know the tool already exists.
func (s *Server) UpdateTool(t Tool) { s.RegisterTool(t) }
