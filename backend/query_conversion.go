package backend

import "context"

// QueryConversionHandler is an EXPERIMENTAL service that allows converting queries between versions

type QueryConversionHandler interface {
	// ConvertQuery is called to covert queries between different versions
	ConvertQuery(ctx context.Context, req *QueryConversionRequest) (*QueryConversionResponse, error)
}

type ConvertQueryFunc func(ctx context.Context, req *QueryConversionRequest) (*QueryConversionResponse, error)

// QueryConversionRequest supports converting a query from on version to another
type QueryConversionRequest struct {
	PluginContext PluginContext `json:"pluginContext"`
	// Queries to convert. This contains the full metadata envelope.
	Query DataQuery `json:"query"`
}

type QueryConversionResponse struct {
	// Converted query. It should extend v0alpha1.Query
	Query any `json:"query"`
	// Result contains extra details into why an conversion request was denied.
	// +optional
	Result *StatusResult `json:"result,omitempty"`
}
