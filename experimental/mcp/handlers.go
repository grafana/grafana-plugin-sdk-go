package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// BindQueryDataHandler attaches the plugin's QueryDataHandler. Schema-driven
// query tools registered via fromschema.RegisterQueryTools delegate to it.
func (s *Server) BindQueryDataHandler(h backend.QueryDataHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queryDataHandler = h
}

// BindCallResourceHandler attaches the plugin's CallResourceHandler. Route
// tools registered via fromschema.RegisterRouteTools delegate to it.
func (s *Server) BindCallResourceHandler(h backend.CallResourceHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callResourceHandler = h
}

// BindCheckHealthHandler attaches the plugin's CheckHealthHandler.
func (s *Server) BindCheckHealthHandler(h backend.CheckHealthHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkHealthHandler = h
}

// QueryDataHandler returns the bound handler (or nil).
func (s *Server) QueryDataHandler() backend.QueryDataHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.queryDataHandler == nil {
		return nil
	}
	return s.queryDataHandler.(backend.QueryDataHandler)
}

func (s *Server) CallResourceHandler() backend.CallResourceHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.callResourceHandler == nil {
		return nil
	}
	return s.callResourceHandler.(backend.CallResourceHandler)
}

func (s *Server) CheckHealthHandler() backend.CheckHealthHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.checkHealthHandler == nil {
		return nil
	}
	return s.checkHealthHandler.(backend.CheckHealthHandler)
}

// executeQueryTool is the shared handler for every query-type tool. It builds
// a single-query QueryDataRequest with the given queryType discriminator and
// the tool args as the JSON body, then JSON-encodes the resulting frames.
func (s *Server) executeQueryTool(ctx context.Context, queryType string, args map[string]any) (any, error) {
	h := s.QueryDataHandler()
	if h == nil {
		return nil, errors.New("no QueryDataHandler bound to MCP server")
	}
	body, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("marshal query args: %w", err)
	}
	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{{
			RefID:     "A",
			QueryType: queryType,
			JSON:      body,
		}},
	}
	resp, err := h.QueryData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("QueryData failed: %w", err)
	}
	out := map[string]any{}
	for refID, dr := range resp.Responses {
		if dr.Error != nil {
			out[refID] = map[string]any{"error": dr.Error.Error()}
			continue
		}
		out[refID] = dr.Frames
	}
	return out, nil
}
