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

	// connect server side first - servers must be connected before clients
	// because the client initializes the MCP session during connection
	if _, err := server.Connect(ctx, serverT, nil); err != nil {
		return nil, nil, err
	}

	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		return nil, nil, err
	}
	return client, session, nil
}
