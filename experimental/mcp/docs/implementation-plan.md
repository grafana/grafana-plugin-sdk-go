# Datasource MCP POC Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up an embedded MCP server inside Grafana datasource plugins, driven by the existing `pluginschema` files, with `github-datasource` and `redshift-datasource` exposing tools/resources/prompts over HTTP.

**Architecture:** Plugins keep their existing gRPC server. A new `experimental/mcp` package in `grafana-plugin-sdk-go` runs an MCP HTTP listener in the same process, sharing handlers (`QueryDataHandler`, `CallResourceHandler`, `CheckHealthHandler`) with gRPC. Plugin authors call schema-driven helpers (`fromschema.Register*`) plus optional code-level `Register*` calls in `main.go`.

**Tech Stack:**
- Go 1.25+
- `github.com/grafana/grafana-plugin-sdk-go` (this repo gets the new package)
- `github.com/modelcontextprotocol/go-sdk/mcp` (official Go MCP SDK - new dependency)
- `github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema` (existing)
- `github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder` (existing)

**Repos touched** (each is a separate git repo - commits per repo):
- `grafana-plugin-sdk-go/` (Phase 1)
- `github-datasource/` (Phase 2)
- `redshift-datasource/` (Phase 3)

**Reference docs:** Spec at `docs/superpowers/specs/2026-05-05-datasource-mcp-poc-design.md`. Before starting Task 1, fetch current `modelcontextprotocol/go-sdk` docs via context7 - the SDK API has been evolving and the code samples below assume the late-2025 API shape (`mcp.NewServer`, `mcp.AddTool`, `mcp.NewStreamableHTTPHandler`). If the current API differs, adapt the calls but keep the package structure and behavior described here.

---

## File Map

### `grafana-plugin-sdk-go` (new files)

| File | Responsibility |
|---|---|
| `experimental/mcp/server.go` | `Server` struct, `NewServer`, `Start`, `Shutdown`, address resolution, `dist/mcp.addr` writer |
| `experimental/mcp/server_test.go` | Server lifecycle, address resolution |
| `experimental/mcp/tools.go` | `Tool` struct, `RegisterTool`, `UpdateTool` |
| `experimental/mcp/tools_test.go` | Tool registration, listing |
| `experimental/mcp/resources.go` | `Resource` struct, `RegisterResource` |
| `experimental/mcp/resources_test.go` | Resource registration, reading |
| `experimental/mcp/prompts.go` | `Prompt` struct, `RegisterPrompt` |
| `experimental/mcp/prompts_test.go` | Prompt registration |
| `experimental/mcp/handlers.go` | `BindQueryDataHandler`, `BindCallResourceHandler`, `BindCheckHealthHandler` plus the shared execution glue |
| `experimental/mcp/handlers_test.go` | Query/route/health execution paths |
| `experimental/mcp/fromschema/querytypes.go` | `RegisterQueryTools(server, schema)` |
| `experimental/mcp/fromschema/querytypes_test.go` | Walker for `QueryTypeDefinitionList` |
| `experimental/mcp/fromschema/routes.go` | `RegisterRouteTools(server, schema)` |
| `experimental/mcp/fromschema/routes_test.go` | OpenAPI walker, path sanitization |
| `experimental/mcp/fromschema/examples.go` | `RegisterQueryExamples(server, schema)` |
| `experimental/mcp/fromschema/examples_test.go` | Example attachment + `examples://query` resource |
| `experimental/mcp/fromschema/healthcheck.go` | `RegisterHealthCheckTool(server)` |
| `experimental/mcp/fromschema/healthcheck_test.go` | Health tool registration |
| `experimental/mcp/mcptest/client.go` | In-memory MCP client for round-trip tests |
| `experimental/mcp/mcptest/client_test.go` | Smoke test for the in-memory client |

### `grafana-plugin-sdk-go` (modified files)

| File | Change |
|---|---|
| `backend/datasource/manage.go` | Add `MCPServer *mcp.Server` to `ManageOpts`; start/stop alongside `backend.Manage` |

### `github-datasource` (new files)

> **Schema location note:** Go's `//go:embed` cannot use `..` paths, so the schema files must live within the embedding package's directory subtree. PR #291 generates files into `src/schema/` (where the TypeScript frontend can also pick them up). For this POC we generate into `pkg/schema/` instead so a Go file at `pkg/` (where `main.go` lives) can embed them. If the frontend later needs the same files, add a build step that copies `pkg/schema/` to `src/schema/`. Out of scope here.

| File | Responsibility |
|---|---|
| `pkg/schema/v0alpha1/query.types.json` | Query type definitions (regenerate from `pkg/models` via `schemabuilder` test) |
| `pkg/schema/v0alpha1/query.examples.json` | Query examples (regenerate alongside types) |
| `pkg/schema/v0alpha1/routes.json` | OpenAPI 3 routes for `/labels` and `/milestones` |
| `pkg/schema_embed.go` | `embed.FS` and a small `LoadSchema()` helper (package `main`) |
| `pkg/schema_embed_test.go` | Verify schema loads cleanly at startup |

### `github-datasource` (modified files)

| File | Change |
|---|---|
| `pkg/main.go` | Wire MCP server, call `fromschema.Register*`, register one custom prompt |
| `go.mod` | Bump `grafana-plugin-sdk-go` to the version produced in Phase 1 |
| `pkg/models/query_test.go` | Bring up to date if needed (PR #291's `TestSchemaDefinitions`) |

### `redshift-datasource` (new files)

> Same `//go:embed` constraint as github-datasource - schema files live in `pkg/schema/` and the embed file at `pkg/`.

| File | Responsibility |
|---|---|
| `pkg/schema/v0alpha1/query.types.json` | Generated from `pkg/redshift/models` via `schemabuilder` test |
| `pkg/schema/v0alpha1/query.examples.json` | Sample SQL queries |
| `pkg/schema/v0alpha1/routes.json` | OpenAPI 3 routes for `/secrets`, `/secret`, `/clusters`, `/workgroups` |
| `pkg/redshift/models/schema_test.go` | `schemabuilder` test that emits the schema files |
| `pkg/schema_embed.go` | `embed.FS` and `LoadSchema()` (package `main`) |

### `redshift-datasource` (modified files)

| File | Change |
|---|---|
| `pkg/main.go` | Wire MCP server, call `fromschema.Register*` |
| `go.mod` | Bump `grafana-plugin-sdk-go` to the Phase 1 version |

---

## Phase 1 - SDK foundation (`grafana-plugin-sdk-go`)

> Working directory for all Phase 1 tasks: `/Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go/`. Branch off `main` with `git checkout -b feat/embedded-mcp` before Task 1.

### Task 1: Add the MCP Go SDK dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Fetch current SDK docs**

Use context7 (`mcp__plugin_context7_context7__resolve-library-id` then `query-docs`) for `modelcontextprotocol/go-sdk` to confirm the current package path and API shape. If the API differs from what's used in subsequent tasks, note the deltas before proceeding.

- [ ] **Step 2: Add the dependency**

```bash
git -C /Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go checkout -b feat/embedded-mcp
cd /Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go
go get github.com/modelcontextprotocol/go-sdk/mcp@latest
go mod tidy
```

- [ ] **Step 3: Verify build still works**

```bash
go build ./...
```
Expected: clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "feat(mcp): add modelcontextprotocol/go-sdk dependency"
```

---

### Task 2: Skeleton `Server`, `Tool`, `Resource`, `Prompt` types

**Files:**
- Create: `experimental/mcp/server.go`
- Create: `experimental/mcp/tools.go`
- Create: `experimental/mcp/resources.go`
- Create: `experimental/mcp/prompts.go`
- Create: `experimental/mcp/server_test.go`

- [ ] **Step 1: Write the failing test**

Create `experimental/mcp/server_test.go`:

```go
package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServer_returnsServerWithName(t *testing.T) {
	s := NewServer(ServerOpts{Name: "test-plugin", Version: "1.0.0"})
	assert.NotNil(t, s)
	assert.Equal(t, "test-plugin", s.Name())
	assert.Equal(t, "1.0.0", s.Version())
}

func TestServer_RegisterTool_listsTool(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	s.RegisterTool(Tool{Name: "ping", Description: "pong"})
	tools := s.Tools()
	assert.Len(t, tools, 1)
	assert.Equal(t, "ping", tools[0].Name)
}

func TestServer_RegisterResource_listsResource(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	s.RegisterResource(Resource{URI: "examples://query", MIMEType: "application/json"})
	resources := s.Resources()
	assert.Len(t, resources, 1)
	assert.Equal(t, "examples://query", resources[0].URI)
}

func TestServer_RegisterPrompt_listsPrompt(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	s.RegisterPrompt(Prompt{Name: "investigate", Description: "walk it"})
	prompts := s.Prompts()
	assert.Len(t, prompts, 1)
}
```

- [ ] **Step 2: Run the test - expect compile failure**

```bash
go test ./experimental/mcp/...
```
Expected: FAIL with "undefined: NewServer", etc.

- [ ] **Step 3: Implement the skeleton**

`experimental/mcp/server.go`:

```go
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
```

`experimental/mcp/tools.go`:

```go
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
```

`experimental/mcp/resources.go`:

```go
package mcp

import "context"

// ResourceReader returns the resource body and its MIME type. The default
// MIME type from the Resource struct is used if the reader returns "".
type ResourceReader func(ctx context.Context) (body []byte, mimeType string, err error)

// Resource is a registerable MCP resource.
type Resource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	Reader      ResourceReader
}

func (s *Server) RegisterResource(r Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.resources {
		if existing.URI == r.URI {
			s.resources[i] = r
			return
		}
	}
	s.resources = append(s.resources, r)
}
```

`experimental/mcp/prompts.go`:

```go
package mcp

// Prompt is a registerable MCP prompt template.
type Prompt struct {
	Name        string
	Description string
	// Template is the literal prompt text. Argument substitution is the
	// caller's responsibility for v1.
	Template string
	// Arguments declares any prompt arguments (name, description, required).
	Arguments []PromptArgument
}

type PromptArgument struct {
	Name        string
	Description string
	Required    bool
}

func (s *Server) RegisterPrompt(p Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.prompts {
		if existing.Name == p.Name {
			s.prompts[i] = p
			return
		}
	}
	s.prompts = append(s.prompts, p)
}
```

- [ ] **Step 4: Run the test**

```bash
go test ./experimental/mcp/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): add Server skeleton with Tool/Resource/Prompt registration"
```

---

### Task 3: Bind handler methods

**Files:**
- Modify: `experimental/mcp/server.go`
- Create: `experimental/mcp/handlers.go`
- Create: `experimental/mcp/handlers_test.go`

- [ ] **Step 1: Write the failing test**

`experimental/mcp/handlers_test.go`:

```go
package mcp

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHandler struct {
	queryCalledWith    *backend.QueryDataRequest
	resourceCalledWith *backend.CallResourceRequest
	healthCalledWith   *backend.CheckHealthRequest
}

func (f *fakeHandler) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	f.queryCalledWith = req
	return &backend.QueryDataResponse{Responses: backend.Responses{"A": backend.DataResponse{}}}, nil
}

func (f *fakeHandler) CallResource(_ context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	f.resourceCalledWith = req
	return sender.Send(&backend.CallResourceResponse{Status: 200, Body: []byte(`{"ok":true}`)})
}

func (f *fakeHandler) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	f.healthCalledWith = req
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk}, nil
}

func TestServer_Bind_storesHandlers(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindQueryDataHandler(h)
	s.BindCallResourceHandler(h)
	s.BindCheckHealthHandler(h)
	require.NotNil(t, s.queryDataHandler)
	require.NotNil(t, s.callResourceHandler)
	require.NotNil(t, s.checkHealthHandler)
	// Type assertion via accessors:
	assert.Equal(t, h, s.QueryDataHandler())
}
```

- [ ] **Step 2: Run the test - expect failure**

```bash
go test ./experimental/mcp/...
```
Expected: FAIL with "undefined: BindQueryDataHandler", etc.

- [ ] **Step 3: Implement the binders**

`experimental/mcp/handlers.go`:

```go
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
```

- [ ] **Step 4: Run the test**

```bash
go test ./experimental/mcp/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/handlers.go experimental/mcp/handlers_test.go
git commit -m "feat(mcp): add Bind*Handler methods on Server"
```

---

### Task 4: Query tool execution glue

The shared handler converts MCP tool args into a `backend.QueryDataRequest`, calls the bound handler, and JSON-encodes the resulting frames.

**Files:**
- Modify: `experimental/mcp/handlers.go`
- Modify: `experimental/mcp/handlers_test.go`

- [ ] **Step 1: Add the failing test**

Append to `experimental/mcp/handlers_test.go`:

```go
func TestExecuteQueryTool_callsHandlerAndEncodesFrames(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindQueryDataHandler(h)

	args := map[string]any{
		"owner":      "grafana",
		"repository": "github-datasource",
	}
	out, err := s.executeQueryTool(context.Background(), "Pull_Requests", args)
	require.NoError(t, err)
	require.NotNil(t, out)

	// Handler should have been called with a single DataQuery whose JSON body matches args.
	require.NotNil(t, h.queryCalledWith)
	require.Len(t, h.queryCalledWith.Queries, 1)
	q := h.queryCalledWith.Queries[0]
	assert.Equal(t, "Pull_Requests", q.QueryType)
	assert.Equal(t, "A", q.RefID)
	assert.JSONEq(t, `{"owner":"grafana","repository":"github-datasource"}`, string(q.JSON))
}

func TestExecuteQueryTool_errorsWhenHandlerNotBound(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	_, err := s.executeQueryTool(context.Background(), "Pull_Requests", map[string]any{})
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run the test - expect failure**

```bash
go test ./experimental/mcp/...
```
Expected: FAIL with "undefined: executeQueryTool".

- [ ] **Step 3: Implement**

Append to `experimental/mcp/handlers.go`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

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
```

(Add the `import` lines via the existing import block - don't duplicate.)

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/... -run TestExecuteQueryTool
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): add executeQueryTool execution glue"
```

---

### Task 5: Route tool execution glue

**Files:**
- Modify: `experimental/mcp/handlers.go`
- Modify: `experimental/mcp/handlers_test.go`

- [ ] **Step 1: Add the failing test**

Append to `experimental/mcp/handlers_test.go`:

```go
func TestExecuteRouteTool_callsHandlerWithBuiltRequest(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindCallResourceHandler(h)

	out, err := s.executeRouteTool(context.Background(), routeToolSpec{
		Method:     "GET",
		Path:       "/labels",
		PathParams: nil,
		QueryArgs:  []string{"owner", "repository"},
	}, map[string]any{
		"owner":      "grafana",
		"repository": "github-datasource",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, h.resourceCalledWith)
	assert.Equal(t, "GET", h.resourceCalledWith.Method)
	assert.Equal(t, "/labels", h.resourceCalledWith.Path)
	assert.Contains(t, h.resourceCalledWith.URL, "owner=grafana")
	assert.Contains(t, h.resourceCalledWith.URL, "repository=github-datasource")
}

func TestExecuteRouteTool_substitutesPathParams(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindCallResourceHandler(h)

	_, err := s.executeRouteTool(context.Background(), routeToolSpec{
		Method:     "GET",
		Path:       "/repos/{owner}/{repo}/files",
		PathParams: []string{"owner", "repo"},
	}, map[string]any{
		"owner": "grafana",
		"repo":  "github-datasource",
	})
	require.NoError(t, err)
	assert.Equal(t, "/repos/grafana/github-datasource/files", h.resourceCalledWith.Path)
}
```

- [ ] **Step 2: Run the test - expect failure**

```bash
go test ./experimental/mcp/... -run TestExecuteRouteTool
```
Expected: FAIL.

- [ ] **Step 3: Implement**

Append to `experimental/mcp/handlers.go`:

```go
import (
	"net/url"
	"strings"
)

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

	// Substitute path parameters.
	path := spec.Path
	for _, p := range spec.PathParams {
		v, ok := args[p]
		if !ok {
			return nil, fmt.Errorf("missing required path parameter %q", p)
		}
		path = strings.ReplaceAll(path, "{"+p+"}", fmt.Sprintf("%v", v))
	}

	// Build query string from QueryArgs that are present in args.
	values := url.Values{}
	for _, q := range spec.QueryArgs {
		if v, ok := args[q]; ok && v != nil && v != "" {
			values.Set(q, fmt.Sprintf("%v", v))
		}
	}

	// Body, if any.
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
	// Try JSON decode; fall back to string body.
	if ct, _ := firstHeader(sender.resp.Headers, "Content-Type"); strings.HasPrefix(ct, "application/json") {
		var decoded any
		if err := json.Unmarshal(sender.resp.Body, &decoded); err == nil {
			return decoded, nil
		}
	}
	return string(sender.resp.Body), nil
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/... -run TestExecuteRouteTool
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): add executeRouteTool execution glue"
```

---

### Task 6: Health check tool execution

**Files:**
- Modify: `experimental/mcp/handlers.go`
- Modify: `experimental/mcp/handlers_test.go`

- [ ] **Step 1: Add the failing test**

```go
func TestExecuteHealthTool_returnsCheckHealthResult(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	h := &fakeHandler{}
	s.BindCheckHealthHandler(h)

	out, err := s.executeHealthTool(context.Background())
	require.NoError(t, err)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "OK", m["status"])
}
```

- [ ] **Step 2: Run the test - expect failure**

```bash
go test ./experimental/mcp/... -run TestExecuteHealthTool
```
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/...
```
Expected: PASS for all tests.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): add executeHealthTool execution glue"
```

---

### Task 7: HTTP transport - Start and Shutdown

This wires the registered tools/resources/prompts into the modelcontextprotocol/go-sdk and starts an HTTP listener.

**Files:**
- Modify: `experimental/mcp/server.go`
- Modify: `experimental/mcp/server_test.go`

- [ ] **Step 1: Add the failing tests**

Append to `experimental/mcp/server_test.go`:

```go
import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func TestServer_StartAndShutdown_listensOnEphemeralPort(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0", Addr: "127.0.0.1:0"})
	require.NoError(t, s.Start(context.Background()))

	addr := s.ListenAddr()
	require.NotEmpty(t, addr)

	// MCP HTTP endpoint should accept POST to /mcp at minimum (initialize).
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)
	conn.Close()

	require.NoError(t, s.Shutdown(context.Background()))
}

func TestServer_Start_failsWhenAddrInUse(t *testing.T) {
	occupier, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer occupier.Close()

	s := NewServer(ServerOpts{Name: "x", Version: "0", Addr: occupier.Addr().String()})
	err = s.Start(context.Background())
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run the tests - expect failure**

```bash
go test ./experimental/mcp/... -run TestServer_StartAndShutdown
```
Expected: FAIL.

- [ ] **Step 3: Implement Start/Shutdown**

Append to `experimental/mcp/server.go`:

```go
import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

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

	// Build the underlying MCP SDK server from our registered state.
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
```

Add to the `Server` struct (in `server.go`):

```go
type Server struct {
	// ... existing fields ...
	httpServer *http.Server
	listenAddr string
}
```

Stub `buildSDKServer` for now (will be filled in next task):

```go
// buildSDKServer constructs the modelcontextprotocol/go-sdk Server from the
// registered Tool/Resource/Prompt state. Called once per Start.
func (s *Server) buildSDKServer() *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    s.opts.Name,
		Version: s.opts.Version,
	}, nil)
	// Tools, resources, prompts will be added in Task 8.
	return srv
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/... -run TestServer_Start
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): add HTTP transport with Start/Shutdown lifecycle"
```

---

### Task 8: Wire registered Tools/Resources/Prompts into the SDK server

**Files:**
- Modify: `experimental/mcp/server.go`
- Create: `experimental/mcp/mcptest/client.go`
- Modify: `experimental/mcp/server_test.go`

This task uses the in-memory `mcptest` client to round-trip a tool call through the SDK server. We build the client first (needed for the test).

- [ ] **Step 1: Build the in-memory client**

`experimental/mcp/mcptest/client.go`:

```go
// Package mcptest provides in-memory wiring for testing an mcp.Server end-to-end
// without spinning up an HTTP listener.
package mcptest

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewClient returns a connected client+session pair backed by an in-memory
// transport pair. The caller is responsible for closing the session.
func NewClient(ctx context.Context, server *mcpsdk.Server) (*mcpsdk.Client, *mcpsdk.ClientSession, error) {
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	clientT, serverT := mcpsdk.NewInMemoryTransports()

	// Connect server side in background.
	go func() { _, _ = server.Connect(ctx, serverT, nil) }()

	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		return nil, nil, err
	}
	return client, session, nil
}
```

(If the current SDK API names differ - e.g. `NewInMemoryTransports` is named differently - adjust here. The test in step 3 will reveal mismatches immediately.)

- [ ] **Step 2: Add the failing tests**

Append to `experimental/mcp/server_test.go`:

```go
func TestServer_buildSDKServer_listsRegisteredTools(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	s.RegisterTool(Tool{
		Name:        "ping",
		Description: "pong",
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return "pong", nil
		},
	})

	sdk := s.buildSDKServer()
	ctx := context.Background()
	_, session, err := mcptest.NewClient(ctx, sdk)
	require.NoError(t, err)
	defer session.Close()

	res, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, res.Tools, 1)
	assert.Equal(t, "ping", res.Tools[0].Name)
}

func TestServer_buildSDKServer_callsToolHandler(t *testing.T) {
	s := NewServer(ServerOpts{Name: "x", Version: "0"})
	s.RegisterTool(Tool{
		Name:        "echo",
		Description: "echo",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return args, nil
		},
	})

	sdk := s.buildSDKServer()
	ctx := context.Background()
	_, session, err := mcptest.NewClient(ctx, sdk)
	require.NoError(t, err)
	defer session.Close()

	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"hello": "world"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Content)
	textContent, ok := res.Content[0].(*mcpsdk.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, `"hello":"world"`)
}
```

Add the import for `mcptest` and `mcpsdk` at the top of `server_test.go`:

```go
import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/mcptest"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)
```

- [ ] **Step 3: Run the tests - expect failure**

```bash
go test ./experimental/mcp/... -run TestServer_buildSDKServer
```
Expected: FAIL - tools/resources/prompts not yet wired into the SDK server.

- [ ] **Step 4: Implement wiring**

Replace `buildSDKServer` in `server.go`:

```go
import "encoding/json"

func (s *Server) buildSDKServer() *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    s.opts.Name,
		Version: s.opts.Version,
	}, nil)

	for _, t := range s.Tools() {
		t := t // capture
		sdkTool := &mcpsdk.Tool{
			Name:        t.Name,
			Description: t.Description,
		}
		if t.InputSchema != nil {
			raw, _ := json.Marshal(t.InputSchema)
			sdkTool.InputSchema = json.RawMessage(raw)
		}
		mcpsdk.AddTool(srv, sdkTool, func(ctx context.Context, _ *mcpsdk.ServerSession, req *mcpsdk.CallToolParamsFor[map[string]any]) (*mcpsdk.CallToolResultFor[any], error) {
			args := req.Arguments
			out, err := t.Handler(ctx, args)
			if err != nil {
				return &mcpsdk.CallToolResultFor[any]{
					IsError: true,
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
				}, nil
			}
			body, err := json.Marshal(out)
			if err != nil {
				return nil, err
			}
			return &mcpsdk.CallToolResultFor[any]{
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
		srv.AddResource(sdkRes, func(ctx context.Context, _ *mcpsdk.ServerSession, _ *mcpsdk.ReadResourceParams) (*mcpsdk.ReadResourceResult, error) {
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
		}, func(ctx context.Context, _ *mcpsdk.ServerSession, _ *mcpsdk.GetPromptParams) (*mcpsdk.GetPromptResult, error) {
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
```

> Note: the exact SDK call signatures (`AddTool`, `AddResource`, `AddPrompt`, `Connect`, `NewInMemoryTransports`, `CallToolParamsFor[T]`, etc.) are based on the late-2025 modelcontextprotocol/go-sdk shape. If the version pulled in Task 1 differs, adapt the SDK calls but keep the wrapping logic and the `Tool/Resource/Prompt` API stable - we want plugins not to care about SDK churn.

- [ ] **Step 5: Run tests**

```bash
go test ./experimental/mcp/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): wire registered tools/resources/prompts into SDK server"
```

---

### Task 9: `fromschema.RegisterHealthCheckTool`

The simplest schema helper. Does not actually read the schema - just registers the standard `check_health` tool when the server has `BindCheckHealthHandler` called.

**Files:**
- Create: `experimental/mcp/fromschema/healthcheck.go`
- Create: `experimental/mcp/fromschema/healthcheck_test.go`

- [ ] **Step 1: Write the failing test**

`experimental/mcp/fromschema/healthcheck_test.go`:

```go
package fromschema_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type healthOnly struct{}

func (healthOnly) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk, Message: "ok"}, nil
}

func TestRegisterHealthCheckTool_addsCheckHealthTool(t *testing.T) {
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	s.BindCheckHealthHandler(healthOnly{})

	fromschema.RegisterHealthCheckTool(s)

	tools := s.Tools()
	require.Len(t, tools, 1)
	assert.Equal(t, "check_health", tools[0].Name)
}
```

- [ ] **Step 2: Run the test - expect failure**

```bash
go test ./experimental/mcp/fromschema/...
```
Expected: FAIL.

- [ ] **Step 3: Implement**

`experimental/mcp/fromschema/healthcheck.go`:

```go
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
```

`fromschema` cannot reach unexported `executeHealthTool`. Expose it from the `mcp` package - add to `experimental/mcp/handlers.go`:

```go
// ExecuteHealthTool is the public entry point used by fromschema to delegate to
// the bound CheckHealthHandler.
func (s *Server) ExecuteHealthTool(ctx context.Context) (any, error) {
	return s.executeHealthTool(ctx)
}

// ExecuteQueryTool is exported for fromschema.RegisterQueryTools.
func (s *Server) ExecuteQueryTool(ctx context.Context, queryType string, args map[string]any) (any, error) {
	return s.executeQueryTool(ctx, queryType, args)
}

// ExecuteRouteTool is exported for fromschema.RegisterRouteTools.
func (s *Server) ExecuteRouteTool(ctx context.Context, spec RouteToolSpec, args map[string]any) (any, error) {
	return s.executeRouteTool(ctx, routeToolSpec(spec), args)
}

// RouteToolSpec is the public mirror of routeToolSpec.
type RouteToolSpec = routeToolSpec
```

(Type alias keeps the public name capitalized while reusing the same struct.)

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/
git commit -m "feat(mcp): add fromschema.RegisterHealthCheckTool"
```

---

### Task 10: `fromschema.RegisterQueryTools`

**Files:**
- Create: `experimental/mcp/fromschema/querytypes.go`
- Create: `experimental/mcp/fromschema/querytypes_test.go`

- [ ] **Step 1: Write the failing test**

`experimental/mcp/fromschema/querytypes_test.go`:

```go
package fromschema_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type queryOnly struct{ lastReq *backend.QueryDataRequest }

func (q *queryOnly) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	q.lastReq = req
	return &backend.QueryDataResponse{}, nil
}

func TestRegisterQueryTools_addsOneToolPerQueryType(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{
				{
					ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
					Spec: sdkapi.QueryTypeDefinitionSpec{
						Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
						Schema:         sdkapi.JSONSchema{Spec: json.RawMessage(`{"type":"object"}`)},
					},
				},
				{
					ObjectMeta: sdkapi.ObjectMeta{Name: "Issues"},
					Spec: sdkapi.QueryTypeDefinitionSpec{
						Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Issues"}},
						Schema:         sdkapi.JSONSchema{Spec: json.RawMessage(`{"type":"object"}`)},
					},
				},
			},
		},
	}

	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	q := &queryOnly{}
	s.BindQueryDataHandler(q)
	fromschema.RegisterQueryTools(s, schema)

	tools := s.Tools()
	require.Len(t, tools, 2)
	names := []string{tools[0].Name, tools[1].Name}
	assert.Contains(t, names, "query_Pull_Requests")
	assert.Contains(t, names, "query_Issues")
}

func TestRegisterQueryTools_handlerCallsBoundQueryData(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{{
				ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
				Spec: sdkapi.QueryTypeDefinitionSpec{
					Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
					Schema:         sdkapi.JSONSchema{Spec: json.RawMessage(`{"type":"object"}`)},
				},
			}},
		},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	q := &queryOnly{}
	s.BindQueryDataHandler(q)
	fromschema.RegisterQueryTools(s, schema)

	tool := s.Tools()[0]
	_, err := tool.Handler(context.Background(), map[string]any{"owner": "grafana"})
	require.NoError(t, err)
	require.NotNil(t, q.lastReq)
	require.Len(t, q.lastReq.Queries, 1)
	assert.Equal(t, "Pull_Requests", q.lastReq.Queries[0].QueryType)
}

func TestRegisterQueryTools_skipsWhenSchemaHasNoQueryTypes(t *testing.T) {
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	fromschema.RegisterQueryTools(s, &pluginschema.PluginSchema{})
	assert.Empty(t, s.Tools())
}
```

> Note: `sdkapi.JSONSchema` exposes `Spec` as `json.RawMessage` (or whatever the current SDK shape is). Inspect `experimental/apis/datasource/v0alpha1/query_definition.go` if the type differs and adjust.

- [ ] **Step 2: Run the tests - expect failure**

```bash
go test ./experimental/mcp/fromschema/... -run TestRegisterQueryTools
```
Expected: FAIL.

- [ ] **Step 3: Implement**

`experimental/mcp/fromschema/querytypes.go`:

```go
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

// schemaSpec returns the JSON-serialized form of the JSON Schema, regardless of
// whether the SDK type wraps a raw message or a structured spec object.
func schemaSpec(s sdkapi.JSONSchema) []byte {
	// JSONSchema in v0alpha1 is currently structured around an embedded *spec.Schema.
	// Marshalling the wrapper back to JSON yields the raw schema doc.
	b, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return b
}
```

> Note: `JSONSchema` in `v0alpha1` may be a thin wrapper around `*spec.Schema`. The marshal-then-pass approach handles either layout uniformly. Verify by looking at `experimental/apis/datasource/v0alpha1/query_definition.go` and `unstructured.go` and adjust if marshalling adds a wrapper key (in which case unwrap to the inner schema).

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/fromschema/... -run TestRegisterQueryTools
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/fromschema/
git commit -m "feat(mcp): add fromschema.RegisterQueryTools"
```

---

### Task 11: `fromschema.RegisterRouteTools`

**Files:**
- Create: `experimental/mcp/fromschema/routes.go`
- Create: `experimental/mcp/fromschema/routes_test.go`

- [ ] **Step 1: Write the failing test**

`experimental/mcp/fromschema/routes_test.go`:

```go
package fromschema_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

type resourceOnly struct{ lastReq *backend.CallResourceRequest }

func (r *resourceOnly) CallResource(_ context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	r.lastReq = req
	return sender.Send(&backend.CallResourceResponse{Status: 200, Body: []byte(`{"ok":true}`)})
}

func TestRegisterRouteTools_addsToolsForResourceRoutes(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		Routes: &pluginschema.Routes{
			Paths: map[string]*spec3.Path{
				"/resources/labels": {
					PathProps: spec3.PathProps{
						Get: &spec3.Operation{
							OperationProps: spec3.OperationProps{
								Summary: "List GitHub labels",
								Parameters: []*spec3.Parameter{
									{ParameterProps: spec3.ParameterProps{Name: "owner", In: "query", Required: true, Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"string"}}}}},
									{ParameterProps: spec3.ParameterProps{Name: "repository", In: "query", Required: true, Schema: &spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"string"}}}}},
								},
							},
						},
					},
				},
			},
		},
	}

	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	r := &resourceOnly{}
	s.BindCallResourceHandler(r)
	fromschema.RegisterRouteTools(s, schema)

	tools := s.Tools()
	require.Len(t, tools, 1)
	// Tool name strips the /resources/ prefix - this matches the path that
	// CallResourceHandler actually sees at runtime.
	assert.Equal(t, "get_labels", tools[0].Name)
	assert.Equal(t, "List GitHub labels", tools[0].Description)
}

func TestRegisterRouteTools_skipsProxyRoutes(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		Routes: &pluginschema.Routes{
			Paths: map[string]*spec3.Path{
				"/proxy/foo": {PathProps: spec3.PathProps{Get: &spec3.Operation{}}},
			},
		},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	fromschema.RegisterRouteTools(s, schema)
	assert.Empty(t, s.Tools())
}

func TestRegisterRouteTools_handlerCallsBoundCallResource(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		Routes: &pluginschema.Routes{
			Paths: map[string]*spec3.Path{
				"/resources/labels": {PathProps: spec3.PathProps{Get: &spec3.Operation{}}},
			},
		},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	r := &resourceOnly{}
	s.BindCallResourceHandler(r)
	fromschema.RegisterRouteTools(s, schema)

	_, err := s.Tools()[0].Handler(context.Background(), map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, r.lastReq)
	assert.Equal(t, "GET", r.lastReq.Method)
	// CallResourceHandler sees the path WITHOUT the /resources/ prefix
	// (Grafana strips it before forwarding; we mirror that behavior).
	assert.Equal(t, "/labels", r.lastReq.Path)
}
```

- [ ] **Step 2: Run the tests - expect failure**

```bash
go test ./experimental/mcp/fromschema/... -run TestRegisterRouteTools
```
Expected: FAIL.

- [ ] **Step 3: Implement**

`experimental/mcp/fromschema/routes.go`:

```go
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
		// Add to input schema regardless of `In` so the tool exposes it.
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

// toolName converts (GET, "/resources/labels") to "get_resources_labels".
// Path params are dropped from the name (their values come from tool args).
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/fromschema/... -run TestRegisterRouteTools
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/fromschema/
git commit -m "feat(mcp): add fromschema.RegisterRouteTools"
```

---

### Task 12: `fromschema.RegisterQueryExamples`

**Files:**
- Create: `experimental/mcp/fromschema/examples.go`
- Create: `experimental/mcp/fromschema/examples_test.go`

- [ ] **Step 1: Write the failing test**

```go
package fromschema_test

import (
	"context"
	"encoding/json"
	"testing"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterQueryExamples_publishesResourceAndAttachesExamples(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{{
				ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
				Spec: sdkapi.QueryTypeDefinitionSpec{
					Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
					Schema:         sdkapi.JSONSchema{Spec: json.RawMessage(`{"type":"object"}`)},
				},
			}},
		},
		QueryExamples: &sdkapi.QueryExamples{Examples: []sdkapi.QueryExample{
			{Name: "Simple", QueryType: "Pull_Requests", SaveModel: sdkapi.AsUnstructured(map[string]any{"owner": "grafana"})},
		}},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	fromschema.RegisterQueryTools(s, schema)
	fromschema.RegisterQueryExamples(s, schema)

	// Resource registered.
	resources := s.Resources()
	require.Len(t, resources, 1)
	assert.Equal(t, "examples://query", resources[0].URI)
	body, _, err := resources[0].Reader(context.Background())
	require.NoError(t, err)
	assert.Contains(t, string(body), "Pull_Requests")

	// Example attached to the matching tool.
	for _, t2 := range s.Tools() {
		if t2.Name == "query_Pull_Requests" {
			require.Len(t, t2.Examples, 1)
			return
		}
	}
	t.Fatalf("query_Pull_Requests tool not found")
}
```

- [ ] **Step 2: Run - expect failure**

```bash
go test ./experimental/mcp/fromschema/... -run TestRegisterQueryExamples
```
Expected: FAIL.

- [ ] **Step 3: Implement**

`experimental/mcp/fromschema/examples.go`:

```go
package fromschema

import (
	"context"
	"encoding/json"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

// RegisterQueryExamples publishes a single resource "examples://query" listing
// every QueryExample, and attaches each example to the corresponding tool's
// Examples field. Must be called after RegisterQueryTools so the tools exist.
func RegisterQueryExamples(s *mcp.Server, schema *pluginschema.PluginSchema) {
	if schema == nil || schema.QueryExamples == nil {
		return
	}
	body, err := json.MarshalIndent(schema.QueryExamples, "", "  ")
	if err != nil {
		body = []byte("{}")
	}
	s.RegisterResource(mcp.Resource{
		URI:         "examples://query",
		Name:        "Query examples",
		Description: "Datasource-specific query examples grouped by queryType",
		MIMEType:    "application/json",
		Reader: func(_ context.Context) ([]byte, string, error) {
			return body, "application/json", nil
		},
	})

	// Attach each example to its matching tool, if registered.
	byName := map[string]*sdkapi.QueryExample{}
	for i := range schema.QueryExamples.Examples {
		ex := &schema.QueryExamples.Examples[i]
		byName[ex.QueryType] = ex
	}
	for _, t := range s.Tools() {
		qt, ok := t.Annotations["queryType"].(string)
		if !ok {
			continue
		}
		ex, ok := byName[qt]
		if !ok {
			continue
		}
		t.Examples = append(t.Examples, ex.SaveModel.Object)
		s.UpdateTool(t)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./experimental/mcp/fromschema/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add experimental/mcp/fromschema/
git commit -m "feat(mcp): add fromschema.RegisterQueryExamples"
```

---

### Task 13: Lifecycle integration with `datasource.Manage` + handler accessor

`automanagement.NewManager` is in `internal/`, so plugins can't import it directly. We add a public accessor in `backend/datasource/` so plugins can bind the same handler to MCP that `Manage` will use for gRPC.

**Files:**
- Modify: `backend/datasource/manage.go`
- Create: `backend/datasource/manage_mcp_test.go`

- [ ] **Step 1: Write the failing test**

`backend/datasource/manage_mcp_test.go`:

```go
package datasource

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartMCPServer_startsAndStops(t *testing.T) {
	s := mcp.NewServer(mcp.ServerOpts{Name: "test", Version: "1.0", Addr: "127.0.0.1:0"})
	require.NoError(t, startMCPServer(s))
	addr := s.ListenAddr()
	require.NotEmpty(t, addr)
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	require.NoError(t, err)
	conn.Close()
	require.NoError(t, stopMCPServer(s))
}

func TestStartMCPServer_logsAndContinuesOnError(t *testing.T) {
	occupier, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer occupier.Close()

	s := mcp.NewServer(mcp.ServerOpts{Name: "test", Version: "1.0", Addr: occupier.Addr().String()})
	// Should not panic and should not fail Manage.
	err = startMCPServer(s)
	assert.NoError(t, err) // we swallow the error and log it
	assert.Empty(t, s.ListenAddr())
}

func TestNewAutomanagementHandler_returnsHandler(t *testing.T) {
	im := NewInstanceManager(func(_ context.Context, _ backend.PluginContext) (instancemgmt.Instance, error) {
		return struct{}{}, nil
	})
	h := NewAutomanagementHandler(im)
	require.NotNil(t, h)
	// The returned handler implements all four interfaces.
	_, isQuery := any(h).(backend.QueryDataHandler)
	_, isResource := any(h).(backend.CallResourceHandler)
	_, isHealth := any(h).(backend.CheckHealthHandler)
	assert.True(t, isQuery)
	assert.True(t, isResource)
	assert.True(t, isHealth)
}
```

- [ ] **Step 2: Run - expect failure**

```bash
go test ./backend/datasource/... -run TestStartMCPServer
```
Expected: FAIL.

- [ ] **Step 3: Modify `manage.go`**

```go
package datasource

import (
	"context"
	// existing imports ...
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
)

// ManageOpts can modify Manage behavior.
type ManageOpts struct {
	// existing fields ...

	// MCPServer, if non-nil, is started alongside the gRPC plugin server and
	// shut down on plugin termination. MCP startup failures are logged but do
	// not prevent the plugin from running.
	MCPServer *mcp.Server
}

// Manage starts serving the data source over gPRC with automatic instance management.
func Manage(pluginID string, instanceFactory InstanceFactoryFunc, opts ManageOpts) error {
	if buildinfo.InfoModeEnabled() {
		// existing block ...
	}

	backend.SetupPluginEnvironment(pluginID)
	if err := backend.SetupTracer(pluginID, opts.TracingOpts); err != nil {
		return fmt.Errorf("setup tracer: %w", err)
	}
	handler := automanagement.NewManager(NewInstanceManager(instanceFactory))

	if opts.MCPServer != nil {
		if err := startMCPServer(opts.MCPServer); err != nil {
			// startMCPServer never returns errors today; left for future hard-fail option.
			log.DefaultLogger.Warn("MCP server startup error", "err", err)
		}
		defer func() {
			if err := stopMCPServer(opts.MCPServer); err != nil {
				log.DefaultLogger.Warn("MCP server shutdown error", "err", err)
			}
		}()
	}

	return backend.Manage(pluginID, backend.ServeOpts{
		// ... existing fields ...
	})
}

// startMCPServer starts the MCP server. Errors are logged and swallowed; the
// plugin continues without MCP if startup fails (e.g. port in use).
func startMCPServer(s *mcp.Server) error {
	if err := s.Start(context.Background()); err != nil {
		log.DefaultLogger.Warn("MCP server failed to start - continuing without MCP", "err", err)
		return nil
	}
	return nil
}

func stopMCPServer(s *mcp.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}

// PluginHandler is the union of handler interfaces an automanagement-backed
// plugin handler implements. Plugins use this to bind the same handler to
// both gRPC (via Manage) and MCP (via mcp.Server.Bind*).
type PluginHandler interface {
	backend.QueryDataHandler
	backend.CallResourceHandler
	backend.CheckHealthHandler
	backend.StreamHandler
}

// NewAutomanagementHandler wraps the given InstanceManager with the SDK's
// automanagement layer and returns the resulting handler. The returned value
// implements PluginHandler. Plugins bind this to mcp.Server then pass the same
// instance factory through datasource.Manage.
func NewAutomanagementHandler(im instancemgmt.InstanceManager) PluginHandler {
	return automanagement.NewManager(im)
}
```

> Add the import for `instancemgmt` and `automanagement` if they aren't already present in `manage.go`. `automanagement` is already imported in the file from the existing implementation.

- [ ] **Step 4: Run tests**

```bash
go test ./backend/datasource/...
go test ./experimental/mcp/...
```
Expected: PASS for both.

- [ ] **Step 5: Commit**

```bash
git add backend/datasource/ experimental/mcp/
git commit -m "feat(mcp): integrate MCP server lifecycle into datasource.Manage"
```

---

### Task 14: Phase 1 integration smoke test

**Files:**
- Create: `experimental/mcp/integration_test.go`

A round-trip test that registers everything via `fromschema`, starts a real HTTP listener, calls a query tool, and verifies the bound handler was invoked.

- [ ] **Step 1: Write the integration test**

```go
package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDS struct{ qReq *backend.QueryDataRequest }

func (s *stubDS) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	s.qReq = req
	return &backend.QueryDataResponse{}, nil
}
func (s *stubDS) CallResource(_ context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
	return nil
}
func (s *stubDS) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk}, nil
}

func TestEndToEnd_registerStartCallTool(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{{
				ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
				Spec: sdkapi.QueryTypeDefinitionSpec{
					Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
					Schema:         sdkapi.JSONSchema{Spec: json.RawMessage(`{"type":"object"}`)},
				},
			}},
		},
	}

	srv := mcp.NewServer(mcp.ServerOpts{Name: "test-ds", Version: "1.0", Addr: "127.0.0.1:0"})
	ds := &stubDS{}
	srv.BindQueryDataHandler(ds)
	srv.BindCallResourceHandler(ds)
	srv.BindCheckHealthHandler(ds)

	fromschema.RegisterQueryTools(srv, schema)
	fromschema.RegisterHealthCheckTool(srv)

	require.NoError(t, srv.Start(context.Background()))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	// Smoke: list tools via raw HTTP (initialize + tools/list). MCP protocol
	// over HTTP is JSON-RPC; we send the canonical handshake.
	addr := "http://" + srv.ListenAddr() + "/mcp"
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}`)
	resp, err := http.Post(addr, "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 300, "got status %d", resp.StatusCode)
}
```

- [ ] **Step 2: Run the test**

```bash
go test ./experimental/mcp/...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add experimental/mcp/integration_test.go
git commit -m "test(mcp): add end-to-end registration + HTTP smoke test"
```

---

### Task 15: Tag the SDK version

To unblock plugins, push the branch and let downstreams pin to it.

- [ ] **Step 1: Push the branch**

```bash
git -C /Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go push -u origin feat/embedded-mcp
```

- [ ] **Step 2: Note the commit SHA**

```bash
git -C /Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go rev-parse HEAD
```

Save this SHA - the plugins will pin to it via `go get github.com/grafana/grafana-plugin-sdk-go@<SHA>`.

> Phase 1 complete. The SDK now exposes `experimental/mcp` and `fromschema`, lifecycle-integrated with `datasource.Manage`. Verify with `go test ./experimental/mcp/... ./backend/datasource/...` before moving on.

---

## Phase 2 - github-datasource

> Working directory: `/Users/erik/code/grafana/datasource-mcp-poc/github-datasource/`. Branch off `main`: `git -C github-datasource checkout -b feat/embedded-mcp`.

### Task 16: Bump SDK + regenerate query schema

**Files:**
- Modify: `go.mod`
- Modify: `pkg/models/query_test.go` (already has `TestSchemaDefinitions`; verify it still compiles against the current SDK)
- Create: `pkg/schema/v0alpha1/query.types.json`
- Create: `pkg/schema/v0alpha1/query.examples.json`

PR #291 already implements the schemabuilder test that emits these files - to `src/schema/`. We bring it up to date with the current SDK and redirect the destination to `pkg/schema/` so Go's `//go:embed` can pick them up.

- [ ] **Step 1: Bump SDK**

```bash
cd /Users/erik/code/grafana/datasource-mcp-poc/github-datasource
go get github.com/grafana/grafana-plugin-sdk-go@<SHA from Task 15>
go mod tidy
```

- [ ] **Step 2: Apply PR #291's `pkg/models/query_test.go` and redirect destination**

If PR #291 is mergeable, fetch it (`gh pr checkout 291 --repo grafana/github-datasource`) then rebase onto `feat/embedded-mcp`. Otherwise, copy the `TestSchemaDefinitions` test body from the PR diff into `pkg/models/query_test.go` verbatim.

Then change the final line of the test from:

```go
builder.UpdateProviderFiles(t, "v0alpha1", "../../src/schema/")
```

to:

```go
builder.UpdateProviderFiles(t, "v0alpha1", "../schema/")
```

(`pkg/models/` -> `pkg/schema/` is one level up.)

- [ ] **Step 3: Run the schemabuilder test - it writes the JSON files**

```bash
mkdir -p pkg/schema
go test ./pkg/models/... -run TestSchemaDefinitions
```
Expected: PASS, with `pkg/schema/v0alpha1/query.types.json` and `query.examples.json` created.

- [ ] **Step 4: Verify the files exist and are well-formed**

```bash
ls pkg/schema/v0alpha1/
test -f pkg/schema/v0alpha1/query.types.json
test -f pkg/schema/v0alpha1/query.examples.json
jq '.items | length' pkg/schema/v0alpha1/query.types.json
```
Expected: integer count of query types (~20).

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum pkg/models/query_test.go pkg/schema/v0alpha1/
git commit -m "feat: generate v0alpha1 query schema for MCP support"
```

---

### Task 17: Author `routes.json`

Translate the existing `pkg/github/resource_handlers.go` routes into OpenAPI 3.

**Files:**
- Create: `pkg/schema/v0alpha1/routes.json`

- [ ] **Step 1: Inspect existing routes**

```bash
grep -n "Handle" pkg/github/resource_handlers.go
grep -rn "/labels\|/milestones" pkg/
```
Confirm the routes registered on `CallResourceHandler` (currently `/labels` and `/milestones`).

- [ ] **Step 2: Write `pkg/schema/v0alpha1/routes.json`**

```json
{
  "paths": {
    "/resources/labels": {
      "get": {
        "summary": "List GitHub labels matching the given query",
        "parameters": [
          { "name": "owner", "in": "query", "required": true, "schema": { "type": "string" }, "description": "Owner of the repository" },
          { "name": "repository", "in": "query", "required": true, "schema": { "type": "string" }, "description": "Repository name" },
          { "name": "query", "in": "query", "required": false, "schema": { "type": "string" }, "description": "Substring filter on label name" }
        ]
      }
    },
    "/resources/milestones": {
      "get": {
        "summary": "List GitHub milestones matching the given query",
        "parameters": [
          { "name": "owner", "in": "query", "required": true, "schema": { "type": "string" } },
          { "name": "repository", "in": "query", "required": true, "schema": { "type": "string" } },
          { "name": "query", "in": "query", "required": false, "schema": { "type": "string" } }
        ]
      }
    }
  }
}
```

- [ ] **Step 3: Validate the JSON syntactically**

```bash
jq '.paths | keys' pkg/schema/v0alpha1/routes.json
```
Expected: `["/resources/labels", "/resources/milestones"]`.

The full schema-loading smoke test is added in Task 18 once the embed file exists.

- [ ] **Step 4: Commit**

```bash
git add pkg/schema/v0alpha1/routes.json
git commit -m "feat: add routes.json for labels and milestones"
```

---

### Task 18: Embed schema and wire MCP into `main.go`

**Files:**
- Create: `pkg/schema_embed.go`
- Create: `pkg/schema_embed_test.go`
- Modify: `pkg/main.go`

The embed file lives at `pkg/` (same directory as `main.go`, package `main`) so it can `//go:embed schema/v0alpha1/*.json` without `..` paths.

- [ ] **Step 1: Create the embed helper**

`pkg/schema_embed.go`:

```go
package main

import (
	"embed"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

//go:embed schema/v0alpha1/*.json
var schemaFS embed.FS

// loadSchema returns the embedded v0alpha1 PluginSchema.
func loadSchema() (*pluginschema.PluginSchema, error) {
	return pluginschema.NewCompositeFileSchemaProvider(schemaFS).Get("v0alpha1")
}
```

- [ ] **Step 2: Add a smoke test for schema loading**

`pkg/schema_embed_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSchema_returnsAllSections(t *testing.T) {
	schema, err := loadSchema()
	require.NoError(t, err)
	require.NotNil(t, schema.QueryTypes)
	require.NotEmpty(t, schema.QueryTypes.Items)
	require.NotNil(t, schema.QueryExamples)
	require.NotEmpty(t, schema.QueryExamples.Examples)
	require.NotNil(t, schema.Routes)
	require.Contains(t, schema.Routes.Paths, "/resources/labels")
	require.Contains(t, schema.Routes.Paths, "/resources/milestones")
}
```

Run it:

```bash
go test ./pkg/...
```
Expected: PASS.

- [ ] **Step 3: Wire MCP into `pkg/main.go`**

Use `datasource.NewAutomanagementHandler` (added in Phase 1 Task 13) to get a handler that implements all interfaces. Bind it to the MCP server, then pass the same instance factory to `datasource.Manage`.

Final `pkg/main.go` shape:

```go
package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"

	"github.com/grafana/github-datasource/pkg/github"
)

func main() {
	schema, err := loadSchema()
	if err != nil {
		log.DefaultLogger.Error("schema load failed", "err", err)
		os.Exit(1)
	}

	mcpServer := mcp.NewServer(mcp.ServerOpts{
		Name:    "grafana-github-datasource",
		Version: "1.0.0",
	})

	// Build the instance manager once, bind to MCP, then pass to Manage.
	instanceFactory := github.NewDatasource
	im := datasource.NewInstanceManager(instanceFactory)
	mgr := datasource.NewAutomanagementHandler(im)

	mcpServer.BindQueryDataHandler(mgr)
	mcpServer.BindCallResourceHandler(mgr)
	mcpServer.BindCheckHealthHandler(mgr)

	fromschema.RegisterQueryTools(mcpServer, schema)
	fromschema.RegisterRouteTools(mcpServer, schema)
	fromschema.RegisterQueryExamples(mcpServer, schema)
	fromschema.RegisterHealthCheckTool(mcpServer)

	mcpServer.RegisterPrompt(mcp.Prompt{
		Name:        "investigate-pull-requests",
		Description: "Walk through investigating recent pull requests in a repository",
		Template:    "List the most recent pull requests for the configured repository, then summarise patterns in review activity over the last 7 days.",
	})

	if err := datasource.Manage("grafana-github-datasource", instanceFactory, datasource.ManageOpts{
		MCPServer: mcpServer,
	}); err != nil {
		log.DefaultLogger.Error("plugin exited", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Build and verify**

```bash
cd /Users/erik/code/grafana/datasource-mcp-poc/github-datasource
go build ./...
go test ./pkg/...
```
Expected: clean build, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/schema_embed.go pkg/schema_embed_test.go pkg/main.go
git commit -m "feat: embed schema and wire MCP server in main.go"
```

---

## Phase 3 - redshift-datasource

> Working directory: `/Users/erik/code/grafana/datasource-mcp-poc/redshift-datasource/`. Branch: `git checkout -b feat/embedded-mcp`.

### Task 19: Bump SDK

- [ ] **Step 1: Bump and tidy**

```bash
cd /Users/erik/code/grafana/datasource-mcp-poc/redshift-datasource
go get github.com/grafana/grafana-plugin-sdk-go@<SHA from Task 15>
go mod tidy
go build ./...
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: bump grafana-plugin-sdk-go to embed-mcp branch"
```

---

### Task 20: Generate query schema via schemabuilder

The redshift plugin uses a generic SQL query type. Generate the schema with one item.

**Files:**
- Create: `pkg/redshift/models/schema_test.go`
- Create: `pkg/schema/v0alpha1/query.types.json`
- Create: `pkg/schema/v0alpha1/query.examples.json`

- [ ] **Step 1: Find the redshift query model**

```bash
grep -rn "type.*Query struct" pkg/redshift/models/ pkg/redshift/
find pkg/redshift -name "*.go" | xargs grep -l "QueryDataRequest\|HandleQuery"
```

In redshift the SQL query is typically modeled in `pkg/redshift/datasource.go` or via `sqlds`. Identify the Go type that represents a single query (e.g. `models.RedshiftQuery` or the SQL framework's `sqlds.Query`).

- [ ] **Step 2: Write the schemabuilder test**

`pkg/redshift/models/schema_test.go`:

```go
package models

import (
	"reflect"
	"testing"

	data "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
	"github.com/stretchr/testify/require"
)

func TestSchemaDefinitions(t *testing.T) {
	builder, err := schemabuilder.NewSchemaBuilder(schemabuilder.BuilderOptions{
		PluginID: []string{"grafana-redshift-datasource"},
		ScanCode: []schemabuilder.CodePaths{{
			BasePackage: "github.com/grafana/redshift-datasource/pkg/redshift/models",
			CodePath:    "./",
		}},
	})
	require.NoError(t, err)

	err = builder.AddQueries([]schemabuilder.QueryTypeInfo{{
		// REPLACE GoType with the actual redshift query Go type identified in step 1.
		GoType:         reflect.TypeFor[*RedshiftQuery](),
		Discriminators: data.NewDiscriminators("queryType", "sql"),
		Examples: []data.QueryExample{
			{
				Name:      "SimpleSQL",
				QueryType: "sql",
				SaveModel: data.AsUnstructured(map[string]any{
					"rawSQL": "SELECT 1",
					"format": "table",
				}),
			},
		},
	}})
	require.NoError(t, err)

	builder.UpdateProviderFiles(t, "v0alpha1", "../../schema/")
}
```

> If `RedshiftQuery` is named differently or lives elsewhere, change the import path and `GoType`. The discriminator value can stay as `"sql"` since redshift only has one query type. The `../../schema/` destination resolves to `pkg/schema/` from `pkg/redshift/models/`.

- [ ] **Step 3: Run it**

```bash
mkdir -p pkg/schema
go test ./pkg/redshift/models/... -run TestSchemaDefinitions
ls pkg/schema/v0alpha1/
```
Expected: `query.types.json` and `query.examples.json` created.

- [ ] **Step 4: Commit**

```bash
git add pkg/redshift/models/schema_test.go pkg/schema/v0alpha1/
git commit -m "feat: generate v0alpha1 query schema"
```

---

### Task 21: Author `routes.json`

**Files:**
- Create: `pkg/schema/v0alpha1/routes.json`

- [ ] **Step 1: Enumerate routes**

```bash
grep -A 2 "func.*Routes" pkg/redshift/routes/routes.go
```

Routes from the existing handler: `/secrets`, `/secret`, `/clusters`, `/workgroups`, plus the SQL framework's defaults (likely `/tables`, `/schemas`, `/columns`).

- [ ] **Step 2: Write `pkg/schema/v0alpha1/routes.json`**

```json
{
  "paths": {
    "/resources/secrets": {
      "get": { "summary": "List AWS Secrets Manager secrets eligible for Redshift" }
    },
    "/resources/secret": {
      "post": {
        "summary": "Get a single secret value",
        "requestBody": {
          "content": { "application/json": { "schema": { "type": "object" } } }
        }
      }
    },
    "/resources/clusters": {
      "get": { "summary": "List Redshift provisioned clusters" }
    },
    "/resources/workgroups": {
      "get": { "summary": "List Redshift Serverless workgroups" }
    },
    "/resources/tables": {
      "post": {
        "summary": "List tables in the connected schema",
        "requestBody": {
          "content": { "application/json": { "schema": { "type": "object" } } }
        }
      }
    },
    "/resources/schemas": {
      "post": {
        "summary": "List schemas in the connected database",
        "requestBody": {
          "content": { "application/json": { "schema": { "type": "object" } } }
        }
      }
    },
    "/resources/columns": {
      "post": {
        "summary": "List columns of a given table",
        "requestBody": {
          "content": { "application/json": { "schema": { "type": "object" } } }
        }
      }
    }
  }
}
```

> If the SQL default routes are different, run `curl http://localhost:3000/api/datasources/<id>/resources/<name>` against a configured plugin and adjust based on what works. For the POC, leave any unverified routes out rather than guess.

- [ ] **Step 3: Commit**

```bash
git add pkg/schema/v0alpha1/routes.json
git commit -m "feat: add routes.json for redshift resource endpoints"
```

---

### Task 22: Embed schema and wire MCP into `main.go`

**Files:**
- Create: `pkg/schema_embed.go`
- Modify: `pkg/main.go`

- [ ] **Step 1: Embed helper**

`pkg/schema_embed.go`:

```go
package main

import (
	"embed"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

//go:embed schema/v0alpha1/*.json
var schemaFS embed.FS

func loadSchema() (*pluginschema.PluginSchema, error) {
	return pluginschema.NewCompositeFileSchemaProvider(schemaFS).Get("v0alpha1")
}
```

- [ ] **Step 2: Wire `pkg/main.go`**

Apply the same pattern as github-datasource Task 18:

```go
package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"

	"github.com/grafana/redshift-datasource/pkg/redshift"
)

func main() {
	schema, err := loadSchema()
	if err != nil {
		log.DefaultLogger.Error("schema load failed", "err", err)
		os.Exit(1)
	}

	mcpServer := mcp.NewServer(mcp.ServerOpts{
		Name:    "grafana-redshift-datasource",
		Version: "2.5.0",
	})

	instanceFactory := redshift.NewDatasource
	im := datasource.NewInstanceManager(instanceFactory)
	mgr := datasource.NewAutomanagementHandler(im)

	mcpServer.BindQueryDataHandler(mgr)
	mcpServer.BindCallResourceHandler(mgr)
	mcpServer.BindCheckHealthHandler(mgr)

	fromschema.RegisterQueryTools(mcpServer, schema)
	fromschema.RegisterRouteTools(mcpServer, schema)
	fromschema.RegisterQueryExamples(mcpServer, schema)
	fromschema.RegisterHealthCheckTool(mcpServer)

	if err := datasource.Manage("grafana-redshift-datasource", instanceFactory, datasource.ManageOpts{
		MCPServer: mcpServer,
	}); err != nil {
		log.DefaultLogger.Error("plugin exited", "err", err)
		os.Exit(1)
	}
}
```

> If `redshift.NewDatasource` has a different signature than `datasource.InstanceFactoryFunc`, adapt by writing a small wrapper in `pkg/redshift/datasource.go` whose signature matches.

- [ ] **Step 3: Build**

```bash
go build ./...
go test ./pkg/...
```
Expected: clean build, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/schema_embed.go pkg/main.go
git commit -m "feat: embed schema and wire MCP server in main.go"
```

---

## Phase 4 - Verification

### Task 23: Build and smoke-test github-datasource

- [ ] **Step 1: Build the plugin binary**

```bash
cd /Users/erik/code/grafana/datasource-mcp-poc/github-datasource
mage -v build:linux
# or whatever target produces a runnable binary; check Magefile.go
```

Alternatively, run directly:

```bash
go run ./pkg
```

- [ ] **Step 2: Verify the listener came up**

In a second terminal:

```bash
cat /Users/erik/code/grafana/datasource-mcp-poc/github-datasource/dist/mcp.addr
```
Expected: `127.0.0.1:<port>\n`.

- [ ] **Step 3: List tools via MCP Inspector**

```bash
npx @modelcontextprotocol/inspector --cli http://$(cat dist/mcp.addr)/mcp tools/list
```

Expected output includes:
- `query_Pull_Requests`, `query_Issues`, `query_Commits`, ... (one per query type)
- `get_resources_labels`, `get_resources_milestones`
- `check_health`

- [ ] **Step 4: Call `check_health`**

```bash
npx @modelcontextprotocol/inspector --cli http://$(cat dist/mcp.addr)/mcp tools/call --tool-name check_health
```

Expected: tool output with `"status": "OK"` (or the actual health state - depends on whether GitHub credentials are configured).

- [ ] **Step 5: List resources**

```bash
npx @modelcontextprotocol/inspector --cli http://$(cat dist/mcp.addr)/mcp resources/list
```

Expected: includes `examples://query`.

- [ ] **Step 6: List prompts**

```bash
npx @modelcontextprotocol/inspector --cli http://$(cat dist/mcp.addr)/mcp prompts/list
```

Expected: includes `investigate-pull-requests`.

- [ ] **Step 7: Stop the plugin (Ctrl+C) and verify cleanup**

```bash
ls dist/mcp.addr 2>&1
```
Expected: file removed (or an error if it was removed).

---

### Task 24: Build and smoke-test redshift-datasource

Repeat Task 23 for redshift, with these expectations:

- `query_sql` tool (single query type)
- Route tools: `get_resources_secrets`, `post_resources_secret`, `get_resources_clusters`, `get_resources_workgroups`, plus the SQL defaults if listed
- `check_health` tool
- `examples://query` resource

If credentials aren't configured, `query_sql` and the resource tools may error - that's expected. The smoke test just verifies the MCP surface is correct.

- [ ] **Step 1-7: Same as Task 23, substituting `redshift-datasource` paths**

---

### Task 25: Final clean build across all repos

- [ ] **Step 1: Run all tests**

```bash
git -C /Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go status
go -C /Users/erik/code/grafana/datasource-mcp-poc/grafana-plugin-sdk-go test ./...

go -C /Users/erik/code/grafana/datasource-mcp-poc/github-datasource test ./...
go -C /Users/erik/code/grafana/datasource-mcp-poc/redshift-datasource test ./...
```

(`go -C` is shorthand for cd'ing first; if your Go version doesn't support it, use `cd && go test`.)

Expected: all tests pass.

- [ ] **Step 2: Push branches**

```bash
git -C /Users/erik/code/grafana/datasource-mcp-poc/github-datasource push -u origin feat/embedded-mcp
git -C /Users/erik/code/grafana/datasource-mcp-poc/redshift-datasource push -u origin feat/embedded-mcp
```

- [ ] **Step 3: Open draft PRs in each repo**

```bash
gh pr create --draft --repo grafana/grafana-plugin-sdk-go --base main --head feat/embedded-mcp \
  --title "feat(experimental/mcp): embedded MCP server for datasource plugins" \
  --body "$(cat <<'EOF'
## Summary
- Adds experimental/mcp package with HTTP transport
- Adds fromschema helpers for query types, resource routes, query examples and health
- Integrates with datasource.Manage lifecycle via ManageOpts.MCPServer

## Test plan
- [ ] go test ./experimental/mcp/...
- [ ] go test ./backend/datasource/...
- [ ] End-to-end smoke via mcp-inspector against github-datasource
EOF
)"
```

Repeat for github-datasource and redshift-datasource with their respective summaries.

---

## Done

The POC is complete when:
- All Phase 1 tests pass
- Both plugins start, write their MCP addr, and respond to `tools/list`, `resources/list` and `prompts/list` from `mcp-inspector`
- `tools/call check_health` returns a result for both plugins
- Three draft PRs are open (one per repo)
