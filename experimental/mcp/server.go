// Package mcp embeds an MCP server inside a Grafana datasource plugin process.
// It binds the plugin's existing gRPC handlers (QueryData, CallResource, CheckHealth)
// to MCP tools, exposes resources and prompts, and runs an HTTP transport alongside
// the gRPC server.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// EnvAddr is the env var that overrides the configured Addr (and forces a
	// specific bind address, ignoring auto-pick).
	EnvAddr = "GF_PLUGIN_MCP_ADDR"
	// AddrFile is written next to dist/standalone.txt with host:port when the
	// listener is bound, so external tooling can find it.
	AddrFile = "dist/mcp.addr"
)

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

	// transport state, populated by Start
	httpServer *http.Server
	listenAddr string
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

// resolveAddr applies the priority order: env var > opts.Addr > auto-pick on 127.0.0.1.
func (s *Server) resolveAddr() string {
	if v := os.Getenv(EnvAddr); v != "" {
		return v
	}
	if s.opts.Addr != "" {
		return s.opts.Addr
	}
	return "127.0.0.1:0"
}

// Start binds the listener and serves the MCP HTTP transport. Non-blocking:
// returns once the listener is accepted, or an error if binding failed.
func (s *Server) Start(ctx context.Context) error {
	if s.httpServer != nil {
		return errors.New("MCP server already started")
	}
	addr := s.resolveAddr()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("MCP listen on %s: %w", addr, err)
	}
	s.listenAddr = listener.Addr().String()

	// build the underlying MCP SDK server from our registered state
	sdkServer := s.buildSDKServer()
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return sdkServer }, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)

	s.httpServer = &http.Server{Handler: mux}
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.DefaultLogger.Error("MCP HTTP server stopped with error", "err", err)
		}
	}()

	if err := s.writeAddrFile(); err != nil {
		log.DefaultLogger.Warn("failed to write MCP addr file", "err", err)
	}
	if !isLoopback(s.listenAddr) {
		log.DefaultLogger.Warn("MCP listener bound to non-loopback address; auth must be handled by a gateway", "addr", s.listenAddr)
	}
	log.DefaultLogger.Info("MCP server listening", "addr", s.listenAddr)
	return nil
}

// Shutdown gracefully stops the HTTP server. Caller should pass a deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	err := s.httpServer.Shutdown(ctx)
	s.httpServer = nil
	_ = os.Remove(AddrFile)
	return err
}

// ListenAddr returns the bound address (host:port) or "" if not started.
func (s *Server) ListenAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listenAddr
}

// buildSDKServer constructs the modelcontextprotocol/go-sdk Server from the
// registered Tool/Resource/Prompt state. Called once per Start.
func (s *Server) buildSDKServer() *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    s.opts.Name,
		Version: s.opts.Version,
	}, nil)

	for _, t := range s.Tools() {
		t := t // capture
		schema := t.InputSchema
		if schema == nil {
			// SDK requires a non-nil object schema; default to an empty object schema
			schema = map[string]any{"type": "object"}
		}
		raw, err := json.Marshal(schema)
		if err != nil {
			log.DefaultLogger.Warn("failed to marshal tool input schema, skipping tool", "tool", t.Name, "err", err)
			continue
		}
		sdkTool := &mcpsdk.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: json.RawMessage(raw),
		}
		srv.AddTool(sdkTool, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			var args map[string]any
			if len(req.Params.Arguments) > 0 {
				if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
					return &mcpsdk.CallToolResult{
						IsError: true,
						Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
					}, nil
				}
			}
			out, err := t.Handler(ctx, args)
			if err != nil {
				return &mcpsdk.CallToolResult{
					IsError: true,
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
				}, nil
			}
			body, err := json.Marshal(out)
			if err != nil {
				return nil, err
			}
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(body)}},
			}, nil
		})
	}

	for _, r := range s.Resources() {
		r := r
		sdkRes := &mcpsdk.Resource{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MIMEType:    r.MIMEType,
		}
		srv.AddResource(sdkRes, func(ctx context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			body, mimeType, err := r.Reader(ctx)
			if err != nil {
				return nil, err
			}
			if mimeType == "" {
				mimeType = r.MIMEType
			}
			return &mcpsdk.ReadResourceResult{
				Contents: []*mcpsdk.ResourceContents{{
					URI:      r.URI,
					MIMEType: mimeType,
					Text:     string(body),
				}},
			}, nil
		})
	}

	for _, p := range s.Prompts() {
		p := p
		args := make([]*mcpsdk.PromptArgument, 0, len(p.Arguments))
		for _, a := range p.Arguments {
			args = append(args, &mcpsdk.PromptArgument{
				Name:        a.Name,
				Description: a.Description,
				Required:    a.Required,
			})
		}
		srv.AddPrompt(&mcpsdk.Prompt{
			Name:        p.Name,
			Description: p.Description,
			Arguments:   args,
		}, func(ctx context.Context, _ *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
			return &mcpsdk.GetPromptResult{
				Messages: []*mcpsdk.PromptMessage{{
					Role:    "user",
					Content: &mcpsdk.TextContent{Text: p.Template},
				}},
			}, nil
		})
	}

	return srv
}

func (s *Server) writeAddrFile() error {
	if err := os.MkdirAll(filepath.Dir(AddrFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(AddrFile, []byte(s.listenAddr+"\n"), 0o644)
}

func isLoopback(hostPort string) bool {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return false
	}
	if host == "" || host == "127.0.0.1" || host == "::1" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
