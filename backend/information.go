package backend

import (
	"context"
	"net/http"
)

const (
	// EndpointSchema friendly name for the schema endpoint/handler.
	EndpointSchema Endpoint = "schema"
)

// SchemaHandler enables users to request data source schemas
type SchemaHandler interface {
	Schema(ctx context.Context, req *SchemaRequest) (*SchemaResponse, error)
}

// SchemaHandlerFunc is an adapter to allow the use of
// ordinary functions as [SchemaHandler]. If f is a function
// with the appropriate signature, SchemaHandlerFunc(f) is a
// [SchemaHandler] that calls f.
type SchemaHandlerFunc func(ctx context.Context, req *SchemaRequest) (*SchemaResponse, error)

// Schema calls fn(ctx, req).
func (fn SchemaHandlerFunc) Schema(ctx context.Context, req *SchemaRequest) (*SchemaResponse, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	return fn(ctx, req)
}

func validateRequest(req *SchemaRequest) error {
	// Validate Type
	if req.Type != "" && req.Type != "tables" && req.Type != "columns" && req.Type != "values" {
		return DownstreamErrorf("Invalid schema request type. Must be one of tables, columns, values")
	}

	if req.Type == "columns" && len(req.Tables) == 0 {
		return DownstreamErrorf("Tables must be specified when requesting columns")
	}

	if req.Type == "values" {
		if len(req.Columns) == 0 {
			return DownstreamErrorf("Columns must be specified when requesting values")
		}
	}

	return nil
}

// SchemaRequest contains the schema request
type SchemaRequest struct {
	// PluginContext the contextual information for the request.
	PluginContext PluginContext

	// Headers the environment/metadata information for the request.
	// To access forwarded HTTP headers please use GetHTTPHeaders or GetHTTPHeader.
	Headers map[string]string

	// Type of data requested. Can be schema | tables | columns | values
	// If empty, defaults to schema.
	// Column requests requires tables to be specified
	// Value requests requires columns to be specified
	Type string

	Tables  []string
	Columns []ColumnsSchemaRequest
}

type ColumnsSchemaRequest struct {
	Table      string
	Parameters map[string]string
}

type Schema struct {
	Tables []Table
	// For the future
	Functions []string
	// Sub table values are listed here as they may be shared across tables
	// We can use the top-level value for enumeration
	SubTableValues map[string]map[string][]string
}

type Table struct {
	Name      string
	SubTables []SubTable
	Columns   []Column
}

type SubTable struct {
	// Sub table name, used to retrieve values from the root SubTableValues map
	Name      string
	DependsOn []string
	// Root is the property used to denote a subtable as top-level
	// Root properties should not have any DependsOn values
	Root bool
}

type Column struct {
	Name string
	Type ColumnType
}

type ColumnType string

const (
	ColumnTypeNumber   ColumnType = "number"
	ColumnTypeString   ColumnType = "string"
	ColumnTypeDatetime ColumnType = "datetime"
)

// CheckHealthResult contains the healthcheck response
type SchemaResponse struct {
	FullSchema   Schema
	Tables       []string
	Columns      map[string][]Column
	ColumnValues map[string][]string
}

// SetHTTPHeader sets the header entries associated with key to the
// single element value. It replaces any existing values
// associated with key. The key is case-insensitive; it is
// canonicalized by textproto.CanonicalMIMEHeaderKey.
func (req *SchemaRequest) SetHTTPHeader(key, value string) {
	if req.Headers == nil {
		req.Headers = map[string]string{}
	}

	setHTTPHeaderInStringMap(req.Headers, key, value)
}

// DeleteHTTPHeader deletes the values associated with key.
// The key is case-insensitive; it is canonicalized by
// CanonicalHeaderKey.
func (req *SchemaRequest) DeleteHTTPHeader(key string) {
	deleteHTTPHeaderInStringMap(req.Headers, key)
}

// GetHTTPHeader gets the first value associated with the given key. If
// there are no values associated with the key, Get returns "".
// It is case-insensitive; textproto.CanonicalMIMEHeaderKey is
// used to canonicalize the provided key. Get assumes that all
// keys are stored in canonical form.
func (req *SchemaRequest) GetHTTPHeader(key string) string {
	return req.GetHTTPHeaders().Get(key)
}

// GetHTTPHeaders returns HTTP headers.
func (req *SchemaRequest) GetHTTPHeaders() http.Header {
	return getHTTPHeadersFromStringMap(req.Headers)
}

var _ ForwardHTTPHeaders = (*SchemaRequest)(nil)
