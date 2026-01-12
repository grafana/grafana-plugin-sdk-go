# Query Schema Introspection - Implementation Plan

This document provides a step-by-step implementation plan for Proposal 1 (New Optional Interface with gRPC Service) from the [Query Schema Introspection design document](./query-schema-introspection.md).

## Overview

We are adding a new optional `QuerySchemaHandler` interface that allows datasource plugins to expose JSON Schema descriptions of their query models. This enables AI agents to discover query structure programmatically.

## Prerequisites

- Familiarity with the [design document](./query-schema-introspection.md)
- Understanding of how existing optional handlers work (e.g., `AdmissionHandler`, `ConversionHandler`, `QueryConversionHandler`)
- Go and Protocol Buffers tooling installed

## Reference Files

These existing files demonstrate the patterns to follow:

| Purpose | Reference File |
|---------|----------------|
| Interface definition | `backend/query_conversion.go` |
| Proto service definition | `proto/backend.proto` (see `AdmissionControl` service) |
| SDK adapter | `backend/admission_adapter.go` |
| gRPC plugin | `backend/grpcplugin/grpc_admission.go` |
| ServeOpts wiring | `backend/serve.go` |
| Proto-to-Go conversion | `backend/convert_from_protobuf.go`, `backend/convert_to_protobuf.go` |

## Implementation Steps

### Step 1: Define the Protocol Buffer Messages and Service

**File to modify:** `proto/backend.proto`

Add the following at the end of the file (before any closing braces):

```protobuf
//-----------------------------------------------
// Query Schema - provides introspection for AI tooling
//-----------------------------------------------

service QuerySchema {
  // GetQuerySchema returns a JSON Schema describing the query model
  rpc GetQuerySchema(GetQuerySchemaRequest) returns (GetQuerySchemaResponse);
}

message GetQuerySchemaRequest {
  PluginContext pluginContext = 1;
  
  // Optional query type to get schema for.
  // If empty, returns the default/primary schema.
  string queryType = 2;
}

message GetQuerySchemaResponse {
  // JSON Schema document (draft-07 or later)
  bytes schema = 1;
  
  // Available query types if the datasource supports multiple
  repeated QueryTypeInfo queryTypes = 2;
}

message QueryTypeInfo {
  // Type identifier (matches DataQuery.QueryType)
  string type = 1;
  
  // Human-readable name
  string name = 2;
  
  // Description of what this query type does
  string description = 3;
}
```

**Then regenerate the protobuf code:**

```bash
cd proto
buf generate
```

This will update `genproto/pluginv2/backend.pb.go` and `genproto/pluginv2/backend_grpc.pb.go`.

### Step 2: Define the Go Interface and Types

**File to create:** `backend/query_schema.go`

```go
package backend

import (
	"context"
	"encoding/json"
)

// QuerySchemaHandler provides JSON Schema introspection for query models.
// This is an optional interface that plugins can implement to enable
// AI-assisted query building and other tooling.
type QuerySchemaHandler interface {
	// GetQuerySchema returns a JSON Schema describing the expected structure
	// of the DataQuery.JSON field for this datasource.
	GetQuerySchema(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error)
}

// GetQuerySchemaRequest is the request for GetQuerySchema.
type GetQuerySchemaRequest struct {
	// PluginContext the contextual information for the request.
	PluginContext PluginContext

	// QueryType allows requesting schema for a specific query type.
	// If empty, returns the default/primary schema.
	QueryType string
}

// GetQuerySchemaResponse is the response from GetQuerySchema.
type GetQuerySchemaResponse struct {
	// Schema is a JSON Schema document (draft-07 or later recommended)
	Schema json.RawMessage

	// QueryTypes lists available query types if the datasource supports multiple.
	// Each type may have a different schema.
	QueryTypes []QueryTypeInfo
}

// QueryTypeInfo describes an available query type.
type QueryTypeInfo struct {
	// Type identifier (matches DataQuery.QueryType)
	Type string

	// Name is a human-readable name for the query type
	Name string

	// Description of what this query type does
	Description string
}

// QuerySchemaHandlerFunc is an adapter to allow the use of ordinary functions
// as QuerySchemaHandler.
type QuerySchemaHandlerFunc func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error)

// GetQuerySchema calls fn(ctx, req).
func (fn QuerySchemaHandlerFunc) GetQuerySchema(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
	return fn(ctx, req)
}
```

### Step 3: Add Protobuf Conversion Functions

**File to modify:** `backend/convert_from_protobuf.go`

Add these methods to the `ConvertFromProtobuf` type:

```go
func (f ConvertFromProtobuf) GetQuerySchemaRequest(req *pluginv2.GetQuerySchemaRequest) *GetQuerySchemaRequest {
	return &GetQuerySchemaRequest{
		PluginContext: f.PluginContext(req.PluginContext),
		QueryType:     req.QueryType,
	}
}
```

**File to modify:** `backend/convert_to_protobuf.go`

Add these methods to the `ConvertToProtobuf` type:

```go
func (t ConvertToProtobuf) GetQuerySchemaResponse(resp *GetQuerySchemaResponse) *pluginv2.GetQuerySchemaResponse {
	if resp == nil {
		return nil
	}
	
	queryTypes := make([]*pluginv2.QueryTypeInfo, len(resp.QueryTypes))
	for i, qt := range resp.QueryTypes {
		queryTypes[i] = &pluginv2.QueryTypeInfo{
			Type:        qt.Type,
			Name:        qt.Name,
			Description: qt.Description,
		}
	}
	
	return &pluginv2.GetQuerySchemaResponse{
		Schema:     resp.Schema,
		QueryTypes: queryTypes,
	}
}
```

### Step 4: Create the SDK Adapter

**File to create:** `backend/query_schema_adapter.go`

```go
package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// querySchemaSDKAdapter adapts the SDK QuerySchemaHandler to the gRPC interface.
type querySchemaSDKAdapter struct {
	handler QuerySchemaHandler
}

func newQuerySchemaSDKAdapter(handler QuerySchemaHandler) *querySchemaSDKAdapter {
	return &querySchemaSDKAdapter{
		handler: handler,
	}
}

func (a *querySchemaSDKAdapter) GetQuerySchema(ctx context.Context, req *pluginv2.GetQuerySchemaRequest) (*pluginv2.GetQuerySchemaResponse, error) {
	parsedReq := FromProto().GetQuerySchemaRequest(req)
	resp, err := a.handler.GetQuerySchema(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().GetQuerySchemaResponse(resp), nil
}
```

### Step 5: Create the gRPC Plugin

**File to create:** `backend/grpcplugin/grpc_query_schema.go`

```go
package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// QuerySchemaServer represents a query schema server.
type QuerySchemaServer interface {
	pluginv2.QuerySchemaServer
}

// QuerySchemaClient represents a query schema client.
type QuerySchemaClient interface {
	pluginv2.QuerySchemaClient
}

// QuerySchemaGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type QuerySchemaGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	QuerySchemaServer QuerySchemaServer
}

// GRPCServer registers p as a query schema gRPC server.
func (p *QuerySchemaGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterQuerySchemaServer(s, &querySchemaGRPCServer{
		server: p.QuerySchemaServer,
	})
	return nil
}

// GRPCClient returns c as a query schema gRPC client.
func (p *QuerySchemaGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &querySchemaGRPCClient{client: pluginv2.NewQuerySchemaClient(c)}, nil
}

type querySchemaGRPCServer struct {
	server QuerySchemaServer
}

func (s *querySchemaGRPCServer) GetQuerySchema(ctx context.Context, req *pluginv2.GetQuerySchemaRequest) (*pluginv2.GetQuerySchemaResponse, error) {
	return s.server.GetQuerySchema(ctx, req)
}

type querySchemaGRPCClient struct {
	client pluginv2.QuerySchemaClient
}

func (m *querySchemaGRPCClient) GetQuerySchema(ctx context.Context, req *pluginv2.GetQuerySchemaRequest, opts ...grpc.CallOption) (*pluginv2.GetQuerySchemaResponse, error) {
	return m.client.GetQuerySchema(ctx, req, opts...)
}

var _ QuerySchemaServer = &querySchemaGRPCServer{}
var _ QuerySchemaClient = &querySchemaGRPCClient{}
```

### Step 6: Update ServeOpts and Wiring

**File to modify:** `backend/grpcplugin/serve.go`

Add to the `ServeOpts` struct:

```go
type ServeOpts struct {
	// ... existing fields ...
	QuerySchemaServer QuerySchemaServer
	// ... rest of fields ...
}
```

Add to the `Serve` function's plugin set creation:

```go
if opts.QuerySchemaServer != nil {
	pSet["querySchema"] = &QuerySchemaGRPCPlugin{
		QuerySchemaServer: opts.QuerySchemaServer,
	}
}
```

**File to modify:** `backend/serve.go`

Add to the `ServeOpts` struct:

```go
type ServeOpts struct {
	// ... existing fields ...
	
	// QuerySchemaHandler provides schema introspection for AI tooling.
	// Optional to implement.
	QuerySchemaHandler QuerySchemaHandler
	
	// ... rest of fields ...
}
```

Update the `GRPCServeOpts` function to wire up the handler:

```go
func GRPCServeOpts(opts ServeOpts) (grpcplugin.ServeOpts, error) {
	// ... existing code ...

	if opts.QuerySchemaHandler != nil {
		pluginOpts.QuerySchemaServer = newQuerySchemaSDKAdapter(opts.QuerySchemaHandler)
	}

	return pluginOpts, nil
}
```

Update `GracefulStandaloneServe` and `TestStandaloneServe` to register the server:

```go
if pluginOpts.QuerySchemaServer != nil {
	pluginv2.RegisterQuerySchemaServer(server, pluginOpts.QuerySchemaServer)
	plugKeys = append(plugKeys, "querySchema")
}
```

### Step 7: Add Tests

**File to create:** `backend/query_schema_test.go`

```go
package backend

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuerySchemaHandlerFunc(t *testing.T) {
	called := false
	handler := QuerySchemaHandlerFunc(func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
		called = true
		return &GetQuerySchemaResponse{
			Schema: json.RawMessage(`{"type": "object"}`),
			QueryTypes: []QueryTypeInfo{
				{Type: "metrics", Name: "Metrics", Description: "Query metrics"},
			},
		}, nil
	})

	resp, err := handler.GetQuerySchema(context.Background(), &GetQuerySchemaRequest{
		QueryType: "metrics",
	})

	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, resp)
	require.Len(t, resp.QueryTypes, 1)
	require.Equal(t, "metrics", resp.QueryTypes[0].Type)
}
```

**File to create:** `backend/query_schema_adapter_test.go`

Add integration tests for the adapter, similar to existing adapter tests like `backend/admission_adapter_test.go`.

### Step 8: Update Documentation

**File to modify:** `README.md` (if applicable)

Document the new optional handler and how plugin authors can implement it.

**Consider creating:** `docs/query-schema.md`

A guide for plugin developers on:
- How to implement `QuerySchemaHandler`
- JSON Schema best practices
- Example implementations
- How to handle multiple query types

## Verification Checklist

After implementation, verify:

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `mage lint` passes (or equivalent linting)
- [ ] Proto regeneration completes without errors
- [ ] A simple test plugin can implement the handler and be called successfully
- [ ] The handler is truly optional (existing plugins still work without changes)

## Example Plugin Implementation

For reference, here's how a plugin author would use this feature:

```go
package main

import (
	"context"
	"encoding/json"
	
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type MyDatasource struct {
	// ... existing fields ...
}

func (d *MyDatasource) GetQuerySchema(ctx context.Context, req *backend.GetQuerySchemaRequest) (*backend.GetQuerySchemaResponse, error) {
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"queryText": map[string]interface{}{
				"type":        "string",
				"description": "The query expression",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
				"default":     100,
			},
		},
		"required": []string{"queryText"},
	}
	
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	
	return &backend.GetQuerySchemaResponse{
		Schema: schemaJSON,
		QueryTypes: []backend.QueryTypeInfo{
			{Type: "", Name: "Default", Description: "Standard query"},
		},
	}, nil
}

func main() {
	ds := &MyDatasource{}
	
	err := backend.Manage("my-plugin", backend.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
		QuerySchemaHandler: ds, // New optional handler
	})
	if err != nil {
		panic(err)
	}
}
```

## Future Considerations

After initial implementation, consider:

1. **SDK helpers for schema generation** - utilities to generate JSON Schema from Go structs
2. **Schema caching** - Grafana-side caching to avoid repeated calls
3. **Validation helper** - optional `ValidateQuery` method using the schema
4. **Example queries** - `GetQueryExamples` for AI few-shot prompting