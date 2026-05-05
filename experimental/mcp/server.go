// Package mcp embeds an MCP server inside a Grafana datasource plugin process.
// It binds the plugin's existing gRPC handlers (QueryData, CallResource, CheckHealth)
// to MCP tools, exposes resources and prompts, and runs an HTTP transport alongside
// the gRPC server.
package mcp

import "sync"

// ServerOpts configures a Server.
type ServerOpts struct {
	Name    string
	Version string
	// Addr is the bind address (e.g. ":7401"). If empty, see address resolution
	// rules in Server.Start: env var first, then auto-pick on 127.0.0.1.
	Addr string
}

// Server is the embedded MCP server for a plugin.
type Server struct {
	opts      ServerOpts
	mu        sync.RWMutex
	tools     []Tool
	resources []Resource
	prompts   []Prompt

	// handler bindings, populated by Bind*
	queryDataHandler    any // backend.QueryDataHandler - typed import in handlers.go
	callResourceHandler any // backend.CallResourceHandler
	checkHealthHandler  any // backend.CheckHealthHandler
}

// NewServer constructs an unstarted Server.
func NewServer(opts ServerOpts) *Server {
	return &Server{opts: opts}
}

func (s *Server) Name() string    { return s.opts.Name }
func (s *Server) Version() string { return s.opts.Version }

// Tools returns a snapshot of registered tools.
func (s *Server) Tools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Tool, len(s.tools))
	copy(out, s.tools)
	return out
}

// Resources returns a snapshot of registered resources.
func (s *Server) Resources() []Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Resource, len(s.resources))
	copy(out, s.resources)
	return out
}

// Prompts returns a snapshot of registered prompts.
func (s *Server) Prompts() []Prompt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Prompt, len(s.prompts))
	copy(out, s.prompts)
	return out
}
