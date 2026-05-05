package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

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

// routeToolSpec describes how to translate tool args into a CallResourceRequest.
// It is built once per tool by fromschema.RegisterRouteTools and reused on
// every call.
type routeToolSpec struct {
	Method     string
	Path       string   // OpenAPI-style path, may contain {param}
	PathParams []string // names of path parameters in Path, e.g. {"owner","repo"}
	QueryArgs  []string // tool arg names that go into the query string
	BodyArg    string   // tool arg name (if any) whose value becomes the request body
}

// captureSender collects the first CallResourceResponse and is enough for v1.
type captureSender struct{ resp *backend.CallResourceResponse }

func (c *captureSender) Send(r *backend.CallResourceResponse) error {
	if c.resp == nil {
		c.resp = r
	}
	return nil
}

func (s *Server) executeRouteTool(ctx context.Context, spec routeToolSpec, args map[string]any) (any, error) {
	h := s.CallResourceHandler()
	if h == nil {
		return nil, errors.New("no CallResourceHandler bound to MCP server")
	}

	// substitute path parameters
	path := spec.Path
	for _, p := range spec.PathParams {
		v, ok := args[p]
		if !ok {
			return nil, fmt.Errorf("missing required path parameter %q", p)
		}
		path = strings.ReplaceAll(path, "{"+p+"}", fmt.Sprintf("%v", v))
	}

	// build query string from QueryArgs that are present in args
	values := url.Values{}
	for _, q := range spec.QueryArgs {
		if v, ok := args[q]; ok && v != nil && v != "" {
			values.Set(q, fmt.Sprintf("%v", v))
		}
	}

	// body, if any
	var body []byte
	if spec.BodyArg != "" {
		if v, ok := args[spec.BodyArg]; ok {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("marshal request body: %w", err)
			}
			body = b
		}
	}

	urlStr := path
	if encoded := values.Encode(); encoded != "" {
		urlStr = path + "?" + encoded
	}

	req := &backend.CallResourceRequest{
		Method: spec.Method,
		Path:   path,
		URL:    urlStr,
		Body:   body,
	}
	sender := &captureSender{}
	if err := h.CallResource(ctx, req, sender); err != nil {
		return nil, fmt.Errorf("CallResource failed: %w", err)
	}
	if sender.resp == nil {
		return nil, errors.New("CallResource returned no response")
	}
	// try JSON decode; fall back to string body
	if ct, _ := firstHeader(sender.resp.Headers, "Content-Type"); strings.HasPrefix(ct, "application/json") {
		var decoded any
		if err := json.Unmarshal(sender.resp.Body, &decoded); err == nil {
			return decoded, nil
		}
	}
	return string(sender.resp.Body), nil
}

func (s *Server) executeHealthTool(ctx context.Context) (any, error) {
	h := s.CheckHealthHandler()
	if h == nil {
		return nil, errors.New("no CheckHealthHandler bound to MCP server")
	}
	res, err := h.CheckHealth(ctx, &backend.CheckHealthRequest{})
	if err != nil {
		return nil, fmt.Errorf("CheckHealth failed: %w", err)
	}
	out := map[string]any{
		"status":  res.Status.String(),
		"message": res.Message,
	}
	if len(res.JSONDetails) > 0 {
		var details any
		if err := json.Unmarshal(res.JSONDetails, &details); err == nil {
			out["details"] = details
		}
	}
	return out, nil
}

func firstHeader(h map[string][]string, key string) (string, bool) {
	if h == nil {
		return "", false
	}
	for k, v := range h {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0], true
		}
	}
	return "", false
}
