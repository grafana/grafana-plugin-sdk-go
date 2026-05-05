package mcp

import "github.com/grafana/grafana-plugin-sdk-go/backend"

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
