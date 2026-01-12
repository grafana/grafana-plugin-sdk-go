package backend

import (
	"context"
	"encoding/json"
)

// EndpointGetQuerySchema friendly name for the get query schema endpoint/handler.
const EndpointGetQuerySchema Endpoint = "getQuerySchema"

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
	// PluginContext is the contextual information for the request.
	PluginContext PluginContext

	// QueryType allows requesting schema for a specific query type.
	// If empty, returns the default/primary schema.
	QueryType string
}

// GetQuerySchemaResponse is the response from GetQuerySchema.
type GetQuerySchemaResponse struct {
	// Schema is a JSON Schema document (draft-07 or later recommended).
	Schema json.RawMessage

	// QueryTypes lists available query types if the datasource supports multiple.
	// Each type may have a different schema.
	QueryTypes []QueryTypeInfo
}

// QueryTypeInfo describes an available query type.
type QueryTypeInfo struct {
	// Type is the identifier for the query type (matches DataQuery.QueryType).
	Type string

	// Name is a human-readable name for the query type.
	Name string

	// Description describes what this query type does.
	Description string
}

// QuerySchemaHandlerFunc is an adapter to allow the use of ordinary functions
// as QuerySchemaHandler.
type QuerySchemaHandlerFunc func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error)

// GetQuerySchema calls fn(ctx, req).
func (fn QuerySchemaHandlerFunc) GetQuerySchema(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
	return fn(ctx, req)
}
