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
