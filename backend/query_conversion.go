package backend

import (
	"context"
)

// QueryConversionHandler is an EXPERIMENTAL service that allows converting queries between versions

type QueryConversionHandler interface {
	// ConvertQuery is called to covert queries between different versions
	ConvertQuery(context.Context, *QueryConversionRequest) (*QueryConversionResponse, error)
}

type ConvertQueryFunc func(context.Context, *QueryConversionRequest) (*QueryConversionResponse, error)

// ConvertQuery calls fn(ctx, req).
func (fn ConvertQueryFunc) ConvertQuery(ctx context.Context, req *QueryConversionRequest) (*QueryConversionResponse, error) {
	return fn(ctx, req)
}

// QueryConversionRequest supports converting a query from on version to another
type QueryConversionRequest struct {
	// NOTE: this may not include app or datasource instance settings depending on the request
	PluginContext PluginContext `json:"pluginContext,omitempty"`
	// Queries to convert. This contains the full metadata envelope.
	Queries []DataQuery `json:"objects,omitempty"`
}

type QueryConversionResponse struct {
	// Converted queries.
	Queries []DataQuery `json:"objects,omitempty"`
}
