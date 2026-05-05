package mcp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
