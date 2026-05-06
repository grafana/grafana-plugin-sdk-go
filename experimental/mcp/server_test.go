package mcp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/mcptest"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeJSONSchema_stripsDraft04Metadata(t *testing.T) {
	in := map[string]any{
		"$schema": "https://json-schema.org/draft-04/schema",
		"id":      "legacy-id",
		"type":    "object",
		"properties": map[string]any{
			"x": map[string]any{
				"$schema": "https://json-schema.org/draft-04/schema",
				"type":    "string",
			},
		},
	}
	out, ok := normalizeJSONSchema(in).(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, out, "$schema")
	assert.NotContains(t, out, "id")
	props := out["properties"].(map[string]any)
	assert.NotContains(t, props["x"].(map[string]any), "$schema")
	// original input must remain untouched
	assert.Contains(t, in, "$schema")
}

func TestNormalizeJSONSchema_convertsExclusiveBoolToNumeric(t *testing.T) {
	in := map[string]any{
		"type":             "number",
		"minimum":          1.0,
		"exclusiveMinimum": true,
		"maximum":          10.0,
		"exclusiveMaximum": false,
	}
	out, ok := normalizeJSONSchema(in).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1.0, out["exclusiveMinimum"])
	assert.NotContains(t, out, "minimum")
	// exclusiveMaximum: false drops both the boolean and keeps maximum as-is
	assert.NotContains(t, out, "exclusiveMaximum")
	assert.Equal(t, 10.0, out["maximum"])
}

func TestNormalizeJSONSchema_leavesNumericExclusiveAlone(t *testing.T) {
	in := map[string]any{
		"type":             "number",
		"exclusiveMinimum": 5.0,
	}
	out, ok := normalizeJSONSchema(in).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 5.0, out["exclusiveMinimum"])
}

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
